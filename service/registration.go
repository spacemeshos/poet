package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
)

type RegisterResult struct {
	Round    string
	RoundEnd time.Time
}

type RoundInfo struct {
	ID    string
	Epoch uint
}

type Registration interface {
	Register(ctx context.Context, nodeID, challenge []byte) (*RegisterResult, error)

	CloseOpenRound(ctx context.Context) (*round, error)

	OpenRound() *round
	ExecutingRound() *round
}

type Registry struct {
	mu        sync.Mutex
	open      *round
	executing *round
	cfg       Config
	dataDir   string
}

func NewRegistry(ctx context.Context, cfg *Config, clock Clock, dataDir string) (reg *Registry, err error) {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0o700); err != nil {
			return nil, err
		}
	}

	open, executing, err := recover(ctx, cfg, dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to recover: %w", err)
	}

	reg = &Registry{
		cfg:       *cfg,
		dataDir:   dataDir,
		open:      open,
		executing: executing,
	}

	if reg.open == nil {
		epoch := uint(0)
		if d := clock.Now().Sub(cfg.Genesis.Time().Add(cfg.PhaseShift)); d > 0 {
			epoch = uint(d/cfg.EpochDuration) + 1
		}
		reg.open, err = reg.newRound(ctx, epoch)
		if err != nil {
			return nil, err
		}
	}

	return reg, nil
}

func (r *Registry) Register(ctx context.Context, nodeID, challenge []byte) (*RegisterResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.open.submit(nodeID, challenge)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{
		Round:    r.open.ID,
		RoundEnd: r.roundEndTime(r.open),
	}, nil
}

func (r *Registry) CloseOpenRound(ctx context.Context) (*round, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	closedRound := r.open

	newRound, err := r.newRound(ctx, closedRound.Epoch()+1)
	if err != nil {
		return nil, err
	}
	r.open = newRound
	r.executing = closedRound
	return closedRound, nil
}

// OpenRound returns the currently open round.
func (r *Registry) OpenRound() *round {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.open
}

// ExecutingRound returns the currently executing round.
func (r *Registry) ExecutingRound() *round {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.executing
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.open != nil {
		err := r.open.teardown(false)
		if err != nil {
			logging.FromContext(context.Background()).Error("failed to close open round", zap.Error(err))
		}
	}
	if r.executing != nil {
		err := r.executing.teardown(false)
		if err != nil {
			logging.FromContext(context.Background()).Error("failed to close executing round", zap.Error(err))
		}
	}

	return nil
}

// newRound creates a new round for the next epoch.
func (r *Registry) newRound(ctx context.Context, epoch uint) (*round, error) {
	newRound, err := newRound(r.dataDir, epoch, r.cfg.MaxRoundMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new round: %w", err)
	}

	if err := newRound.saveState(); err != nil {
		_ = newRound.teardown(true)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	logging.FromContext(ctx).Info("Round opened", zap.String("ID", newRound.ID))
	return newRound, nil
}

func (r *Registry) roundStartTime(round *round) time.Time {
	return r.cfg.Genesis.Time().Add(r.cfg.PhaseShift).Add(r.cfg.EpochDuration * time.Duration(round.Epoch()))
}

func (r *Registry) roundEndTime(round *round) time.Time {
	return r.roundStartTime(round).Add(r.cfg.EpochDuration).Add(-r.cfg.CycleGap)
}

func recover(ctx context.Context, cfg *Config, dataDir string) (open *round, executing *round, err error) {
	logger := logging.FromContext(ctx).Named("recovery")
	logger.Info("Recovering rounds state", zap.String("datadir", dataDir))
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		logger.Sugar().Infof("recovering entry %s", entry.Name())
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}
		r, err := newRound(dataDir, uint(epoch), cfg.MaxRoundMembers)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create round: %w", err)
		}

		err = r.loadState()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid round state: %w", err)
		}

		if r.isOpen() {
			logger.Info("found round in open state.", zap.String("ID", r.ID))
			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			open = r
			continue
		}

		logger.Info("found round in executing state.", zap.String("ID", r.ID))
		if executing != nil {
			logger.Warn("found more than 1 executing round - overwriting", zap.String("previous", executing.ID))
		}
		executing = r
	}

	return open, executing, nil
}
