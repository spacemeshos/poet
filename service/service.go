package service

import (
	"context"
	"crypto/ed25519"
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

	"github.com/spacemeshos/poet/config/round_config"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/service/tid"
	"github.com/spacemeshos/poet/shared"
)

//go:generate mockgen -package mocks -destination mocks/service.go . RegistrationService

type Service struct {
	registration RegistrationService

	genesis        time.Time
	cfg            Config
	roundCfg       round_config.Config
	datadir        string
	minMemoryLayer uint

	privKey ed25519.PrivateKey
}

type ClosedRound struct {
	Epoch          uint
	MembershipRoot []byte
}

type RegistrationService interface {
	RegisterForRoundClosed(ctx context.Context) <-chan ClosedRound
	NewProof(ctx context.Context, proof shared.NIP) error
}

type newServiceOption struct {
	privateKey ed25519.PrivateKey
	cfg        Config
	roundCfg   round_config.Config
}

type newServiceOptionFunc func(*newServiceOption)

func WithPrivateKey(privateKey ed25519.PrivateKey) newServiceOptionFunc {
	return func(o *newServiceOption) {
		o.privateKey = privateKey
	}
}

func WithConfig(cfg Config) newServiceOptionFunc {
	return func(o *newServiceOption) {
		o.cfg = cfg
	}
}

func WithRoundConfig(cfg round_config.Config) newServiceOptionFunc {
	return func(o *newServiceOption) {
		o.roundCfg = cfg
	}
}

// NewService creates a new instance of Poet Service.
// It should be started with `Service::Run`.
func NewService(
	ctx context.Context,
	genesis time.Time,
	datadir string,
	registration RegistrationService,
	opts ...newServiceOptionFunc,
) (*Service, error) {
	options := &newServiceOption{
		cfg:      DefaultConfig(),
		roundCfg: round_config.DefaultConfig(),
	}
	for _, opt := range opts {
		opt(options)
	}
	if options.privateKey == nil {
		_, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("generating key: %w", err)
		}
		options.privateKey = privateKey
	}

	estimatedLeaves := uint64(
		options.roundCfg.EpochDuration.Seconds()-options.roundCfg.CycleGap.Seconds(),
	) * uint64(
		options.cfg.EstimatedLeavesPerSecond,
	)
	minMemoryLayer := uint(0)
	if totalLayers := mshared.RootHeightFromWidth(estimatedLeaves); totalLayers > options.cfg.MemoryLayers {
		minMemoryLayer = totalLayers - options.cfg.MemoryLayers
	}

	logger := logging.FromContext(ctx)
	logger.Sugar().Infof("creating poet service. min memory layer: %v. genesis: %v", minMemoryLayer, genesis.String())
	logger.Info(
		"epoch configuration",
		zap.Duration("duration", options.roundCfg.EpochDuration),
		zap.Duration("cycle gap", options.roundCfg.CycleGap),
		zap.Duration("phase shift", options.roundCfg.PhaseShift),
	)

	roundsDir := filepath.Join(datadir, "rounds")
	if _, err := os.Stat(roundsDir); os.IsNotExist(err) {
		if err := os.Mkdir(roundsDir, 0o700); err != nil {
			return nil, err
		}
	}

	s := &Service{
		genesis:        genesis,
		cfg:            options.cfg,
		roundCfg:       options.roundCfg,
		minMemoryLayer: minMemoryLayer,
		datadir:        datadir,
		privKey:        options.privateKey,
		registration:   registration,
	}

	logging.FromContext(ctx).Sugar().Infof("service public key: %x", s.privKey.Public().(ed25519.PublicKey))

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
		err := round.recoverExecution(ctx, end, s.cfg.TreeFileBufferSize)
		unlock()
		switch {
		case err == nil:
			s.onNewProof(ctx, round.epoch, round.execution)
		case errors.Is(err, context.Canceled):
			logger.Info("recovered round execution canceled", zap.Uint("epoch", round.epoch))
		default:
			logger.Error("recovered round execution failed", zap.Error(err), zap.Uint("epoch", round.epoch))
		}
		eg.Go(func() error {
			if err := round.teardown(ctx, err == nil); err != nil {
				logger.Warn("round teardown failed", zap.Error(err))
			}
			return nil
		})
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
			if s.roundCfg.RoundEnd(s.genesis, closedRound.Epoch).Before(time.Now()) {
				logger.Info("skipping past round")
				continue
			}

			round, err := newRound(
				filepath.Join(s.datadir, "rounds"),
				closedRound.Epoch,
				withMembershipRoot(closedRound.MembershipRoot),
			)
			if err != nil {
				return fmt.Errorf("failed to create a new round: %w", err)
			}

			unlock := lockOSThread(ctx, roundTidFile)
			err = round.execute(
				logging.NewContext(ctx, logger),
				s.roundCfg.RoundEnd(s.genesis, round.epoch),
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
				if err := round.teardown(ctx, err == nil); err != nil {
					logger.Warn("round teardown failed", zap.Error(err))
				}
				return nil
			})

		case <-ctx.Done():
			logger.Info("service shutting down")
			return eg.Wait()
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
	logger := logging.FromContext(ctx).Named("recovery")
	logger.Info("recovering service state", zap.String("datadir", s.datadir))
	entries, err := os.ReadDir(roundsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		logger.Sugar().Infof("recovering entry %s", entry.Name())
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}
		r, err := newRound(roundsDir, uint(epoch))
		if err != nil {
			return nil, fmt.Errorf("failed to create round: %w", err)
		}

		err = r.loadState()
		if err != nil {
			return nil, fmt.Errorf("invalid round state: %w", err)
		}

		logger.Info("recovered round", zap.Uint("epoch", r.epoch))

		switch {
		case r.isExecuted():
			logger.Info(
				"round is finished already",
				zap.Time("started", r.executionStarted),
				zap.Binary("root", r.execution.NIP.Root),
				zap.Uint64("num_leaves", r.execution.NumLeaves),
			)
			s.onNewProof(ctx, r.epoch, r.execution)
			r.teardown(ctx, true)

		default:
			// Round is in executing state.
			logger.Info(
				"round is executing",
				zap.Time("started", r.executionStarted),
				zap.Uint64("num_leaves", r.execution.NumLeaves),
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
	s.registration.NewProof(ctx, shared.NIP{
		MerkleProof: *execution.NIP,
		Leaves:      execution.NumLeaves,
		Epoch:       epoch,
	})
}
