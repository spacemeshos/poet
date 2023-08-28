package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
	}, []string{"id"})

	leavesMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "leaves_total",
		Help:      "Number of generated leaves in a round",
	}, []string{"id"})

	batchWriteLatencyMetric = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "batch_write_latency_seconds",
		Help:      "Latency of batch write operations",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 20),
	})
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

type pendingSubmit struct {
	done      chan<- error
	challenge []byte
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

	// protects concurrent access to batch and pendingSubmits
	// (both are used in submit and flushPendingSubmits) while
	// the latter can be ran on a goroutine.
	batchMutex     sync.Mutex
	batch          *leveldb.Batch
	pendingSubmits map[string]pendingSubmit
	flushInterval  time.Duration
	maxBatchSize   int
}

func (r *round) Epoch() uint32 {
	return r.epoch
}

type newRoundOptions struct {
	submitFlushInterval time.Duration
	maxMembers          uint
	maxSubmitBatchSize  int
}

type newRoundOptionFunc func(*newRoundOptions)

func withMaxMembers(maxMembers uint) newRoundOptionFunc {
	return func(o *newRoundOptions) {
		o.maxMembers = maxMembers
	}
}

func withSubmitFlushInterval(interval time.Duration) newRoundOptionFunc {
	return func(o *newRoundOptions) {
		o.submitFlushInterval = interval
	}
}

func withMaxSubmitBatchSize(size int) newRoundOptionFunc {
	return func(o *newRoundOptions) {
		o.maxSubmitBatchSize = size
	}
}

func newRound(dbdir, datadir string, epoch uint32, options ...newRoundOptionFunc) (*round, error) {
	opts := newRoundOptions{
		submitFlushInterval: time.Microsecond,
		maxMembers:          1 << 32,
		maxSubmitBatchSize:  1000,
	}
	for _, opt := range options {
		opt(&opts)
	}
	id := strconv.FormatUint(uint64(epoch), 10)
	datadir = filepath.Join(datadir, id)
	dbdir = filepath.Join(dbdir, id)

	db, err := leveldb.OpenFile(dbdir, nil)
	if err != nil {
		return nil, fmt.Errorf("opening challenges DB: %w", err)
	}

	if err := os.MkdirAll(datadir, 0o700); err != nil {
		return nil, fmt.Errorf("creating round datadir: %w", err)
	}

	// Note: using the panicking version here because it panics
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
		maxMembers:     opts.maxMembers,
		membersCounter: membersCounter,
		leavesCounter:  leavesCounter,
		maxBatchSize:   opts.maxSubmitBatchSize,
		flushInterval:  opts.submitFlushInterval,
	}

	membersCounter.Add(float64(r.members))

	return r, nil
}

// submit a challenge to the round under the given key.
// The challenges are collected in a batch and persisted to disk periodically.
// Returns an error if it was not possible to add the challenge to the batch (for example, duplicated key)
// and a channel to which the final result will be sent when the challenge is persisted.
// The caller must await the returned channel to make sure that the challenge is persisted.
func (r *round) submit(ctx context.Context, key, challenge []byte) (<-chan error, error) {
	if !r.isOpen() {
		return nil, ErrRoundIsNotOpen
	}

	r.batchMutex.Lock()
	defer r.batchMutex.Unlock()

	if r.members+uint(len(r.pendingSubmits)) >= r.maxMembers {
		return nil, ErrMaxMembersReached
	}

	if r.batch == nil {
		r.pendingSubmits = make(map[string]pendingSubmit)
		r.batch = leveldb.MakeBatch(r.maxBatchSize)
		time.AfterFunc(r.flushInterval, r.flushPendingSubmits)
	}

	pending, ok := r.pendingSubmits[string(key)]
	if ok {
		switch {
		case bytes.Equal(challenge, pending.challenge):
			return nil, fmt.Errorf("%w: key: %X", ErrChallengeAlreadySubmitted, key)
		default:
			return nil, ErrConflictingRegistration
		}
	}

	registered, err := r.challengesDb.Get(key, nil)
	switch {
	case errors.Is(err, leveldb.ErrNotFound):

		// OK - challenge is not registered yet.
	case err != nil:
		return nil, fmt.Errorf("failed to check if challenge is registered: %w", err)
	case bytes.Equal(challenge, registered):
		return nil, fmt.Errorf("%w: key: %X", ErrChallengeAlreadySubmitted, key)
	default:
		return nil, ErrConflictingRegistration
	}

	r.batch.Put(key, challenge)
	done := make(chan error, 1)
	r.pendingSubmits[string(key)] = pendingSubmit{
		done,
		challenge,
	}

	if r.batch.Len() >= r.maxBatchSize {
		r.flushPendingSubmitsLocked()
	}

	return done, nil
}

func (r *round) flushPendingSubmits() {
	r.batchMutex.Lock()
	defer r.batchMutex.Unlock()
	r.flushPendingSubmitsLocked()
}

func (r *round) flushPendingSubmitsLocked() {
	if r.batch == nil {
		return
	}
	logging.FromContext(context.Background()).
		Debug("flushing pending submits", zap.Int("num", len(r.pendingSubmits)), zap.String("round", r.ID))
	start := time.Now()
	err := r.challengesDb.Write(r.batch, &opt.WriteOptions{Sync: true})
	if err == nil {
		batchWriteLatencyMetric.Observe(time.Since(start).Seconds())
	}
	for _, pending := range r.pendingSubmits {
		pending.done <- err
		close(pending.done)
	}
	r.members += uint(len(r.pendingSubmits))
	r.membersCounter.Add(float64(len(r.pendingSubmits)))

	r.pendingSubmits = nil
	r.batch = nil
}

func (r *round) execute(ctx context.Context, end time.Time, minMemoryLayer, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))

	r.flushPendingSubmits()

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

	logger.With().Info(
		"recovering execution",
		zap.Time("end", end),
		zap.Int("members", len(r.execution.Members)),
		zap.Uint64("num_leaves", r.execution.NumLeaves),
		zap.Array("parked_nodes", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			for _, node := range r.execution.ParkedNodes {
				enc.AppendString(fmt.Sprintf("%X", node))
			}
			return nil
		})),
	)

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

	r.flushPendingSubmits()

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
