package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spacemeshos/merkle-tree/cache"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/state"
)

var leavesMetric = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "poet",
	Subsystem: "round",
	Name:      "leaves_total",
	Help:      "Number of generated leaves in a round",
}, []string{"id"})

// TODO(poszu): align with `executionState` when migrated to new state structure
// (https://github.com/spacemeshos/poet/issues/384).
type persistedRoundState struct {
	SecurityParam uint8
	Members       [][]byte // Deprecated
	Statement     []byte
	ParkedNodes   [][]byte // Deprecated
	NumLeaves     uint64
	NIP           *shared.MerkleProof
}

type executionState struct {
	SecurityParam uint8
	Statement     []byte
	NumLeaves     uint64
	NIP           *shared.MerkleProof
}

const roundStateFileBaseName = "state.bin"

type roundState struct {
	ExecutionStarted time.Time
	Execution        *persistedRoundState
}

func (r *round) IsFinished() bool {
	return r.execution.NIP != nil
}

type round struct {
	epoch            uint
	datadir          string
	ID               string
	executionStarted time.Time
	execution        *executionState

	leavesCounter prometheus.Counter
}

type newRoundOptionFunc func(*round)

func WithMembershipRoot(root []byte) newRoundOptionFunc {
	return func(r *round) {
		r.execution.Statement = root
	}
}

func NewRound(datadir string, epoch uint, opts ...newRoundOptionFunc) (*round, error) {
	id := strconv.FormatUint(uint64(epoch), 10)
	datadir = filepath.Join(datadir, id)

	if err := os.MkdirAll(datadir, 0o700); err != nil {
		return nil, fmt.Errorf("creating round datadir: %w", err)
	}

	// Note: using the panicking version here because it panics
	// only if the number of label values is not the same as the number of variable labels in Desc.
	// There is only 1 label  (round ID), that  is passed, so it's safe to use.
	leavesCounter := leavesMetric.WithLabelValues(id)

	r := &round{
		epoch:   epoch,
		datadir: datadir,
		ID:      id,
		execution: &executionState{
			SecurityParam: shared.T,
		},
		leavesCounter: leavesCounter,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

func (r *round) Execute(ctx context.Context, end time.Time, minMemoryLayer, fileWriterBufSize uint) error {
	r.executionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	logging.FromContext(ctx).Info(
		"executing round",
		zap.Time("end", end),
		zap.Binary("statement", r.execution.Statement),
	)

	numLeaves, nip, err := prover.GenerateProof(
		ctx,
		r.leavesCounter,
		prover.TreeConfig{
			MinMemoryLayer:    minMemoryLayer,
			Datadir:           r.datadir,
			FileWriterBufSize: fileWriterBufSize,
		},
		hash.GenLabelHashFunc(r.execution.Statement),
		hash.GenMerkleHashFunc(r.execution.Statement),
		end,
		r.execution.SecurityParam,
		r.persistExecution,
	)
	if err != nil {
		return fmt.Errorf("generating proof: %w", err)
	}
	r.execution.NumLeaves, r.execution.NIP = numLeaves, nip
	if err := r.saveState(); err != nil {
		return err
	}

	logging.FromContext(ctx).Info(
		"execution ended",
		zap.Binary("root", r.execution.NIP.Root),
		zap.Uint64("num_leaves", r.execution.NumLeaves),
		zap.Duration("duration", time.Since(r.executionStarted)),
	)
	return nil
}

func (r *round) persistExecution(
	ctx context.Context,
	treeCache *cache.Writer,
	numLeaves uint64,
) error {
	logging.FromContext(ctx).
		Info("persisting execution state", zap.Uint64("numLeaves", numLeaves), zap.Uint("epoch", r.epoch))

	// Call GetReader() so that the cache would flush and validate structure.
	if _, err := treeCache.GetReader(); err != nil {
		return err
	}

	r.execution.NumLeaves = numLeaves
	return r.saveState()
}

func (r *round) RecoverExecution(ctx context.Context, end time.Time, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.Uint("epoch", r.epoch))
	logger.With().Info(
		"recovering execution",
		zap.Time("end", end),
		zap.Uint64("num_leaves", r.execution.NumLeaves),
	)

	started := time.Now()
	numLeaves, nip, err := prover.GenerateProofRecovery(
		ctx,
		r.leavesCounter,
		prover.TreeConfig{
			Datadir:           r.datadir,
			FileWriterBufSize: fileWriterBufSize,
		},
		hash.GenLabelHashFunc(r.execution.Statement),
		hash.GenMerkleHashFunc(r.execution.Statement),
		end,
		r.execution.SecurityParam,
		r.execution.NumLeaves,
		r.persistExecution,
	)
	if err != nil {
		return err
	}
	r.execution.NumLeaves, r.execution.NIP = numLeaves, nip
	if err := r.saveState(); err != nil {
		return err
	}

	logger.With().Info(
		"finished round recovered execution",
		zap.Binary("root", r.execution.NIP.Root),
		zap.Uint64("num_leaves", r.execution.NumLeaves),
		zap.Duration("duration", time.Since(started)),
	)

	return nil
}

// loadState recovers persisted state from disk.
func (r *round) loadState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	s := roundState{}
	if err := state.Load(filename, &s); err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	if r.execution.SecurityParam != s.Execution.SecurityParam {
		return errors.New("SecurityParam config mismatch")
	}
	r.execution.NIP = s.Execution.NIP
	r.execution.NumLeaves = s.Execution.NumLeaves
	r.execution.Statement = s.Execution.Statement
	r.execution.SecurityParam = s.Execution.SecurityParam
	r.executionStarted = s.ExecutionStarted

	return nil
}

func (r *round) saveState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	err := state.Persist(filename, &roundState{
		ExecutionStarted: r.executionStarted,
		Execution: &persistedRoundState{
			SecurityParam: r.execution.SecurityParam,
			Statement:     r.execution.Statement,
			NumLeaves:     r.execution.NumLeaves,
			NIP:           r.execution.NIP,
		},
	})
	if err != nil {
		return fmt.Errorf("persisting state: %w", err)
	}
	return nil
}

func (r *round) Teardown(ctx context.Context, cleanup bool) error {
	logger := logging.FromContext(ctx)
	logger.Info("tearing down round", zap.Uint("epoch", r.epoch), zap.Bool("cleanup", cleanup))
	started := time.Now()
	defer logger.Info(
		"finished tearing down round",
		zap.Uint("epoch", r.epoch),
		zap.Duration("duration", time.Since(started)),
	)

	leavesMetric.DeleteLabelValues(r.ID)

	if cleanup {
		return os.RemoveAll(r.datadir)
	}
	return r.saveState()
}
