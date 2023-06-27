package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	Members          uint
}

func (r *round) isOpen() bool {
	return r.executionStarted.IsZero()
}

func (r *round) isExecuted() bool {
	return r.execution.NIP != nil
}

type round struct {
	epoch            uint
	datadir          string
	ID               string
	challengesDb     *leveldb.DB
	executionStarted time.Time
	execution        *executionState
	members          uint
	maxMembers       uint
}

func (r *round) Epoch() uint {
	return r.epoch
}

func newRound(datadir string, epoch uint, maxMembers uint) (*round, error) {
	id := strconv.FormatUint(uint64(epoch), 10)
	datadir = filepath.Join(datadir, id)

	db, err := leveldb.OpenFile(filepath.Join(datadir, "challengesDb"), nil)
	if err != nil {
		_ = os.RemoveAll(datadir)
		return nil, err
	}

	return &round{
		epoch:        epoch,
		datadir:      datadir,
		ID:           id,
		challengesDb: db,
		execution: &executionState{
			SecurityParam: shared.T,
		},
		maxMembers: maxMembers,
	}, nil
}

func (r *round) submit(key, challenge []byte) error {
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
	}
	return err
}

func (r *round) execute(ctx context.Context, end time.Time, minMemoryLayer uint, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))
	logger.Sugar().Infof("executing until %v...", end)

	r.executionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	if members, statement, err := r.calcMembersAndStatement(); err != nil {
		return err
	} else {
		r.execution.Members, r.execution.Statement = members, statement
	}

	if err := r.saveState(); err != nil {
		return err
	}

	numLeaves, nip, err := prover.GenerateProof(
		ctx,
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
		return err
	}
	r.execution.NumLeaves, r.execution.NIP = numLeaves, nip
	if err := r.saveState(); err != nil {
		return err
	}

	logger.Sugar().Infof("execution ended, phi=%x, duration %v", r.execution.NIP.Root, time.Since(r.executionStarted))
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
	if err := r.saveState(); err != nil {
		return err
	}

	return nil
}

func (r *round) recoverExecution(ctx context.Context, end time.Time, fileWriterBufSize uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))
	logger.With().Info("recovering execution", zap.Time("end", end))

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

	numLeaves, nip, err := prover.GenerateProofRecovery(
		ctx,
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

	logger.With().Info("finished round recovered execution", zap.Duration("duration", time.Since(started)))

	return nil
}

// loadState recovers persisted state from disk.
func (r *round) loadState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	state := roundState{}
	if err := load(filename, &state); err != nil {
		return err
	}
	if r.execution.SecurityParam != state.Execution.SecurityParam {
		return errors.New("SecurityParam config mismatch")
	}
	r.execution = state.Execution
	r.executionStarted = state.ExecutionStarted
	r.members = state.Members

	return nil
}

func (r *round) saveState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	return persist(filename, &roundState{
		ExecutionStarted: r.executionStarted,
		Execution:        r.execution,
		Members:          r.members,
	})
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

func (r *round) teardown(cleanup bool) error {
	if err := r.challengesDb.Close(); err != nil {
		return err
	}

	if cleanup {
		return os.RemoveAll(r.datadir)
	}
	return r.saveState()
}
