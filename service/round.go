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
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
)

var (
	ErrRoundIsNotOpen    = errors.New("round is not open")
	ErrMaxMembersReached = errors.New("maximum number of round members reached")

	membersMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "members_total",
		Help:      "Number of members in a round",
	}, []string{"ID"})

	leavesMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "leaves_total",
		Help:      "Number of generated leaves in a round",
	}, []string{"ID"})
)

type executionState struct {
	SecurityParam uint8
	Members       [][]byte
	Statement     []byte
	ParkedNodes   [][]byte
	NumLeaves     uint64
	NIP           *shared.MerkleProof
}

const roundStateFileBaseName = "state.bin"

type roundState struct {
	ExecutionStarted time.Time
	Execution        *executionState
}

func (r *round) isOpen() bool {
	return r.executionStarted.IsZero()
}

func (r *round) isExecuted() bool {
	return r.execution.NIP != nil
}

type round struct {
	epoch            uint32
	datadir          string
	ID               string
	challengesDb     *leveldb.DB
	executionStarted time.Time
	execution        *executionState
	members          uint
	maxMembers       uint

	membersCounter prometheus.Counter
	leavesCounter  prometheus.Counter
}

func (r *round) Epoch() uint32 {
	return r.epoch
}

func newRound(datadir string, epoch uint32, maxMembers uint) (*round, error) {
	id := strconv.FormatUint(uint64(epoch), 10)
	datadir = filepath.Join(datadir, id)

	db, err := leveldb.OpenFile(filepath.Join(datadir, "challengesDb"), nil)
	if err != nil {
		_ = os.RemoveAll(datadir)
		return nil, err
	}

	// Note: using the panicking version of prometheus.NewCounterVec() because it panics
	// only if the number of label values is not the same as the number of variable labels in Desc.
	// There is only 1 label  (round ID), that  is passed, so it's safe to use.
	membersCounter := membersMetric.WithLabelValues(id)
	leavesCounter := leavesMetric.WithLabelValues(id)

	r := &round{
		epoch:        epoch,
		datadir:      datadir,
		ID:           id,
		challengesDb: db,
		execution: &executionState{
			SecurityParam: shared.T,
		},
		members:        countMembersInDB(db),
		maxMembers:     maxMembers,
		membersCounter: membersCounter,
		leavesCounter:  leavesCounter,
	}

	membersCounter.Add(float64(r.members))

	return r, nil
}

func (r *round) submit(ctx context.Context, key, challenge []byte) error {
	if !r.isOpen() {
		return ErrRoundIsNotOpen
	}

	if r.members >= r.maxMembers {
		return ErrMaxMembersReached
	}

	if has, err := r.challengesDb.Has(key, nil); err != nil {
		return err
	} else if has {
		return fmt.Errorf("%w: key: %X", ErrChallengeAlreadySubmitted, key)
	}
	err := r.challengesDb.Put(key, challenge, &opt.WriteOptions{Sync: true})
	if err == nil {
		r.members += 1
		if r.membersCounter != nil {
			r.membersCounter.Inc()
		}
	}
	return err
}

func (r *round) execute(ctx context.Context, end time.Time, minMemoryLayer, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))

	r.executionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	if members, statement, err := r.calcMembersAndStatement(); err != nil {
		return err
	} else {
		r.execution.Members, r.execution.Statement = members, statement
	}

	logger.Info(
		"executing round",
		zap.Time("end", end),
		zap.Int("members", len(r.execution.Members)),
		zap.Binary("statement", r.execution.Statement),
	)

	if err := r.saveState(); err != nil {
		return err
	}

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

	logger.Info(
		"execution ended",
		zap.Binary("root", r.execution.NIP.Root),
		zap.Uint64("num_leaves", r.execution.NumLeaves),
		zap.Duration("duration", time.Since(r.executionStarted)),
	)
	return nil
}

func (r *round) persistExecution(
	ctx context.Context,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	numLeaves uint64,
) error {
	logging.FromContext(ctx).
		Info("persisting execution state", zap.Uint64("numLeaves", numLeaves), zap.String("round", r.ID))

	// Call GetReader() so that the cache would flush and validate structure.
	if _, err := treeCache.GetReader(); err != nil {
		return err
	}

	r.execution.NumLeaves = numLeaves
	r.execution.ParkedNodes = tree.GetParkedNodes(r.execution.ParkedNodes[:0])
	return r.saveState()
}

func (r *round) recoverExecution(ctx context.Context, end time.Time, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))

	started := time.Now()

	if r.execution.Members == nil || r.execution.Statement == nil {
		logger.Debug("calculating members and statement")
		members, statement, err := r.calcMembersAndStatement()
		if err != nil {
			return fmt.Errorf("failed to calculate members and statement")
		}
		r.execution.Members, r.execution.Statement = members, statement
		if err := r.saveState(); err != nil {
			return err
		}
	}

	logger.With().
		Info("recovering execution", zap.Time("end", end), zap.Int("members", len(r.execution.Members)), zap.Uint64("num_leaves", r.execution.NumLeaves))

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
		r.execution.ParkedNodes,
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
	state := roundState{}
	if err := load(filename, &state); err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	if r.execution.SecurityParam != state.Execution.SecurityParam {
		return errors.New("SecurityParam config mismatch")
	}
	r.execution = state.Execution
	r.executionStarted = state.ExecutionStarted

	return nil
}

func (r *round) saveState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	err := persist(filename, &roundState{
		ExecutionStarted: r.executionStarted,
		Execution:        r.execution,
	})
	if err != nil {
		return fmt.Errorf("persisting state: %w", err)
	}
	return nil
}

func (r *round) calcMembersAndStatement() ([][]byte, []byte, error) {
	mtree, err := merkle.NewTreeBuilder().
		WithHashFunc(shared.HashMembershipTreeNode).
		Build()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize merkle tree: %v", err)
	}

	members := make([][]byte, 0)
	iter := r.challengesDb.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		challenge := iter.Value()
		challengeCopy := make([]byte, len(challenge))
		copy(challengeCopy, challenge)

		members = append(members, challengeCopy)
		if err := mtree.AddLeaf(challengeCopy); err != nil {
			return nil, nil, err
		}
	}

	return members, mtree.Root(), nil
}

func (r *round) teardown(ctx context.Context, cleanup bool) error {
	logger := logging.FromContext(ctx)
	logger.Info("tearing down round", zap.String("round", r.ID), zap.Bool("cleanup", cleanup))
	started := time.Now()
	defer logger.Info(
		"finished tearing down round",
		zap.String("round", r.ID),
		zap.Duration("duration", time.Since(started)),
	)

	membersMetric.DeleteLabelValues(r.ID)
	leavesMetric.DeleteLabelValues(r.ID)

	if err := r.challengesDb.Close(); err != nil {
		return fmt.Errorf("closing DB: %w", err)
	}

	if cleanup {
		return os.RemoveAll(r.datadir)
	}
	return r.saveState()
}

func countMembersInDB(db *leveldb.DB) (count uint) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count
}
