package registration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spacemeshos/merkle-tree"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
)

var (
	ErrMaxMembersReached         = errors.New("maximum number of round members reached")
	ErrChallengeAlreadySubmitted = errors.New("challenge is already submitted")
	ErrConflictingRegistration   = errors.New("conflicting registration")

	membersMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "members_total",
		Help:      "Number of members in a round",
	}, []string{"id"})

	batchWriteLatencyMetric = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "poet",
		Subsystem: "round",
		Name:      "batch_write_latency_seconds",
		Help:      "Latency of batch write operations",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 20),
	})
)

type pendingSubmit struct {
	done      chan<- error
	challenge []byte
}

type round struct {
	epoch uint
	db    *leveldb.DB

	maxMembers     int
	members        int
	membersCounter prometheus.Counter

	// protects concurrent access to batch and pendingSubmits
	// (both are used in submit and flushPendingSubmits) while
	// the latter can be ran on a goroutine.
	batchMutex     sync.Mutex
	batch          *leveldb.Batch
	pendingSubmits map[string]pendingSubmit
	flushInterval  time.Duration
	maxBatchSize   int
}

type newRoundOptions struct {
	submitFlushInterval time.Duration
	maxMembers          int
	maxSubmitBatchSize  int
	failIfNotExists     bool
}

type newRoundOptionFunc func(*newRoundOptions)

func withMaxMembers(maxMembers int) newRoundOptionFunc {
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

func failIfNotExists() newRoundOptionFunc {
	return func(o *newRoundOptions) {
		o.failIfNotExists = true
	}
}

func newRound(epoch uint, dbdir string, options ...newRoundOptionFunc) (*round, error) {
	id := strconv.FormatUint(uint64(epoch), 10)
	opts := newRoundOptions{
		submitFlushInterval: time.Microsecond,
		maxMembers:          1 << 32,
		maxSubmitBatchSize:  1000,
	}
	for _, opt := range options {
		opt(&opts)
	}

	dbdir = filepath.Join(dbdir, "rounds", epochToRoundId(epoch))
	db, err := leveldb.OpenFile(dbdir, &opt.Options{ErrorIfMissing: opts.failIfNotExists})
	if err != nil {
		return nil, fmt.Errorf("failed to open round db: %w", err)
	}

	// Note: using the panicking version here because it panics
	// only if the number of label values is not the same as the number of variable labels in Desc.
	// There is only 1 label  (round ID), that  is passed, so it's safe to use.
	membersCounter := membersMetric.WithLabelValues(id)

	r := &round{
		epoch:          epoch,
		db:             db,
		members:        countMembersInDB(db),
		membersCounter: membersCounter,
		maxMembers:     opts.maxMembers,
		maxBatchSize:   opts.maxSubmitBatchSize,
		flushInterval:  opts.submitFlushInterval,
		pendingSubmits: make(map[string]pendingSubmit),
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
	r.batchMutex.Lock()
	defer r.batchMutex.Unlock()

	if r.members+len(r.pendingSubmits) >= r.maxMembers {
		return nil, ErrMaxMembersReached
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

	if r.batch == nil {
		r.batch = leveldb.MakeBatch(r.maxBatchSize)
		time.AfterFunc(r.flushInterval, r.flushPendingSubmits)
	}

	registered, err := r.db.Get(key, nil)
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
	if r.batch == nil || r.batch.Len() == 0 {
		return
	}
	logging.FromContext(context.Background()).
		Debug("flushing pending submits", zap.Int("num", len(r.pendingSubmits)), zap.Uint("round", r.epoch))
	start := time.Now()
	err := r.db.Write(r.batch, &opt.WriteOptions{Sync: true})
	if err == nil {
		batchWriteLatencyMetric.Observe(time.Since(start).Seconds())
	}
	for _, pending := range r.pendingSubmits {
		pending.done <- err
		close(pending.done)
	}
	r.members += len(r.pendingSubmits)
	r.membersCounter.Add(float64(len(r.pendingSubmits)))

	r.pendingSubmits = make(map[string]pendingSubmit)
	r.batch = nil
}

func countMembersInDB(db *leveldb.DB) (count int) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count
}

func (r *round) calcMembershipRoot() ([]byte, error) {
	r.flushPendingSubmits()
	mtree, err := merkle.NewTreeBuilder().
		WithHashFunc(shared.HashMembershipTreeNode).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize merkle tree: %v", err)
	}

	iter := r.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		if err := mtree.AddLeaf(iter.Value()); err != nil {
			return nil, err
		}
	}

	return mtree.Root(), nil
}

func (r *round) getMembers() (members [][]byte) {
	iter := r.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		member := append([]byte{}, iter.Value()...)
		members = append(members, member)
	}
	return members
}

func (r *round) Close() error {
	r.flushPendingSubmits()
	return r.db.Close()
}
