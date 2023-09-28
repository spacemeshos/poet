package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	mshared "github.com/spacemeshos/merkle-tree/shared"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/service/tid"
	"github.com/spacemeshos/poet/shared"
)

//go:generate mockgen -package mocks -destination mocks/service.go . RegistrationService

type Service struct {
	registration RegistrationService

	genesis        time.Time
	cfg            Config
	roundCfg       roundConfig
	datadir        string
	minMemoryLayer uint
}

type ClosedRound struct {
	Epoch          uint
	MembershipRoot []byte
}

type RegistrationService interface {
	RegisterForRoundClosed(ctx context.Context) <-chan ClosedRound
	NewProof(ctx context.Context, proof shared.NIP) error
}

type roundConfig interface {
	RoundEnd(genesis time.Time, epoch uint) time.Time
	RoundDuration() time.Duration
}

type newServiceOption struct {
	cfg Config
}

type newServiceOptionFunc func(*newServiceOption)

func WithConfig(cfg Config) newServiceOptionFunc {
	return func(o *newServiceOption) {
		o.cfg = cfg
	}
}

// New creates a new instance of a worker service.
// It should be started with `Service::Run`.
func New(
	ctx context.Context,
	genesis time.Time,
	datadir string,
	registration RegistrationService,
	roundCfg roundConfig,
	opts ...newServiceOptionFunc,
) (*Service, error) {
	options := &newServiceOption{
		cfg: DefaultConfig(),
	}
	for _, opt := range opts {
		opt(options)
	}

	estimatedLeaves := uint64(roundCfg.RoundDuration().Seconds()) * uint64(options.cfg.EstimatedLeavesPerSecond)
	minMemoryLayer := uint(0)
	if totalLayers := mshared.RootHeightFromWidth(estimatedLeaves); totalLayers > options.cfg.MemoryLayers {
		minMemoryLayer = totalLayers - options.cfg.MemoryLayers
	}

	s := &Service{
		genesis:        genesis,
		cfg:            options.cfg,
		roundCfg:       roundCfg,
		minMemoryLayer: minMemoryLayer,
		datadir:        datadir,
		registration:   registration,
	}

	logging.FromContext(ctx).Info(
		"created poet worker service",
		zap.Time("genesis", s.genesis),
		zap.Uint("min memory layer", s.minMemoryLayer),
	)

	return s, nil
}

func (s *Service) loop(ctx context.Context, roundToResume *round) error {
	logger := logging.FromContext(ctx).Named("worker")
	ctx = logging.NewContext(ctx, logger)

	var eg errgroup.Group
	defer eg.Wait()

	// file that round's thread ID is written to
	roundTidFile := path.Join(s.datadir, "round.tid")

	// Resume recovered round if any
	if round := roundToResume; round != nil {
		end := s.roundCfg.RoundEnd(s.genesis, round.epoch)
		unlock := lockOSThread(ctx, roundTidFile)
		err := round.RecoverExecution(ctx, end, s.cfg.TreeFileBufferSize)
		unlock()
		cleanupRound := func(err error) {
			eg.Go(func() error {
				if err := round.Teardown(ctx, err == nil); err != nil {
					logger.Warn("round teardown failed", zap.Error(err), zap.Uint("epoch", round.epoch))
				}
				return nil
			})
		}
		switch {
		case err == nil:
			s.onNewProof(ctx, round.epoch, round.execution)
			cleanupRound(nil)
		case errors.Is(err, context.Canceled):
			logger.Info("recovered round execution canceled", zap.Uint("epoch", round.epoch))
			cleanupRound(err)
			return nil
		default:
			logger.Error("recovered round execution failed", zap.Error(err), zap.Uint("epoch", round.epoch))
			cleanupRound(err)
			return err
		}
	}

	closedRounds := s.registration.RegisterForRoundClosed(ctx)

	for {
		select {
		case closedRound := <-closedRounds:
			logger := logger.With(zap.Uint("epoch", closedRound.Epoch))
			logger.Info(
				"received round to execute",
				zap.Binary("root", closedRound.MembershipRoot),
			)
			end := s.roundCfg.RoundEnd(s.genesis, closedRound.Epoch)
			if end.Before(time.Now()) {
				logger.Info("skipping past round", zap.Time("expected end", end))
				continue
			}

			round, err := NewRound(
				filepath.Join(s.datadir, "rounds"),
				closedRound.Epoch,
				WithMembershipRoot(closedRound.MembershipRoot),
			)
			if err != nil {
				return fmt.Errorf("failed to create a new round: %w", err)
			}

			unlock := lockOSThread(ctx, roundTidFile)
			err = round.Execute(
				logging.NewContext(ctx, logger),
				end,
				s.minMemoryLayer,
				s.cfg.TreeFileBufferSize,
			)
			unlock()
			switch {
			case err == nil:
				s.onNewProof(ctx, round.epoch, round.execution)
			case errors.Is(err, context.Canceled):
				logger.Info("round canceled")
			default:
				logger.Error("round failed", zap.Error(err))
			}

			eg.Go(func() error {
				if err := round.Teardown(ctx, err == nil); err != nil {
					logger.Warn("round teardown failed", zap.Error(err))
				}
				return nil
			})

		case <-ctx.Done():
			logger.Info("service shutting down")
			return nil
		}
	}
}

// lockOSThread:
// - locks current goroutine to OS thread,
// - writes the current TID to `tidFile`.
// The caller must call the returned `unlock` to unlock OS thread.
func lockOSThread(ctx context.Context, tidFile string) (unlock func()) {
	runtime.LockOSThread()

	if err := os.WriteFile(tidFile, []byte(strconv.Itoa(tid.Gettid())), os.ModePerm); err != nil {
		logging.FromContext(ctx).Warn("failed to write goroutine thread id to file", zap.Error(err))
	}

	return runtime.UnlockOSThread
}

// Run starts the Service's loop.
// It stops when the `ctx` is canceled.
func (s *Service) Run(ctx context.Context) error {
	toResume, err := s.recover(ctx)
	if err != nil {
		return fmt.Errorf("failed to recover: %v", err)
	}

	return s.loop(ctx, toResume)
}

func (s *Service) recover(ctx context.Context) (executing *round, err error) {
	roundsDir := filepath.Join(s.datadir, "rounds")
	entries, err := os.ReadDir(roundsDir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	}

	logger := logging.FromContext(ctx).Named("recovery")
	logger.Info("recovering worker state", zap.String("datadir", s.datadir))

	for _, entry := range entries {
		logger.Sugar().Infof("recovering entry %s", entry.Name())
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("entry is not an uint32 %s", entry.Name())
		}
		r, err := NewRound(roundsDir, uint(epoch))
		if err != nil {
			return nil, fmt.Errorf("failed to create round: %w", err)
		}

		logger.Info("recovered round", zap.Uint("epoch", r.epoch))

		switch {
		case r.IsFinished():
			logger.Info(
				"round is finished already",
				zap.Time("started", r.executionStarted),
				zap.Binary("root", r.execution.NIP.Root),
				zap.Uint64("leaves", r.execution.NumLeaves),
			)
			s.onNewProof(ctx, r.epoch, r.execution)
			r.Teardown(ctx, true)
		case r.executionStarted.IsZero():
			// An open round from a previous poet version
			logger.Info("round is open, removing it", zap.Uint("epoch", r.epoch))
			r.Teardown(ctx, true)
		default:
			logger.Info(
				"round is executing",
				zap.Time("started", r.executionStarted),
				zap.Uint64("leaves", r.execution.NumLeaves),
			)
			if executing != nil {
				logger.Warn("found more than 1 executing round - overwriting", zap.Uint("previous", executing.epoch))
			}
			executing = r
		}
	}

	return executing, nil
}

func (s *Service) onNewProof(ctx context.Context, epoch uint, execution *executionState) {
	logging.FromContext(ctx).
		Info("publishing new proof", zap.Uint("epoch", epoch), zap.Uint64("leaves", execution.NumLeaves))
	if err := s.registration.NewProof(ctx, shared.NIP{
		MerkleProof: *execution.NIP,
		Leaves:      execution.NumLeaves,
		Epoch:       epoch,
	}); err != nil {
		logging.FromContext(ctx).Error("failed to publish new proof", zap.Error(err))
	}
}
