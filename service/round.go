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

var ErrRoundIsNotOpen = errors.New("round is not open")

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

func (r *roundState) isOpen() bool {
	return r.ExecutionStarted.IsZero()
}

func (r *roundState) isExecuted() bool {
	return r.Execution.NIP != nil
}

type round struct {
	epoch   uint32
	datadir string
	ID      string

	challengesDb *leveldb.DB
	roundState
}

func (r *round) Epoch() uint32 {
	return r.epoch
}

func newRound(datadir string, epoch uint32) (*round, error) {
	id := strconv.FormatUint(uint64(epoch), 10)
	datadir = filepath.Join(datadir, id)

	db, err := leveldb.OpenFile(filepath.Join(datadir, "challengesDb"), nil)
	if err != nil {
		return nil, err
	}

	return &round{
		epoch:        epoch,
		datadir:      datadir,
		ID:           id,
		challengesDb: db,
		roundState: roundState{
			Execution: &executionState{
				SecurityParam: shared.T,
			},
		},
	}, nil
}

func (r *round) submit(key, challenge []byte) error {
	if !r.isOpen() {
		return ErrRoundIsNotOpen
	}

	if has, err := r.challengesDb.Has(key, nil); err != nil {
		return err
	} else if has {
		return fmt.Errorf("%w: key: %X", ErrChallengeAlreadySubmitted, key)
	}
	return r.challengesDb.Put(key, challenge, &opt.WriteOptions{Sync: true})
}

func (r *round) execute(ctx context.Context, end time.Time, minMemoryLayer uint) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))
	logger.Sugar().Infof("executing until %v...", end)

	r.ExecutionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	if members, statement, err := r.calcMembersAndStatement(); err != nil {
		return err
	} else {
		r.Execution.Members, r.Execution.Statement = members, statement
	}

	if err := r.saveState(); err != nil {
		return err
	}

	numLeaves, nip, err := prover.GenerateProof(
		ctx,
		r.datadir,
		hash.GenLabelHashFunc(r.Execution.Statement),
		hash.GenMerkleHashFunc(r.Execution.Statement),
		end,
		r.Execution.SecurityParam,
		minMemoryLayer,
		r.persistExecution,
	)
	if err != nil {
		return err
	}
	r.Execution.NumLeaves, r.Execution.NIP = numLeaves, nip
	if err := r.saveState(); err != nil {
		return err
	}

	logger.Sugar().Sugar().Infoff("execution ended, phi=%x, duration %v", r.Execution.NIP.Root, time.Since(r.ExecutionStarted))
	return nil
}

func (r *round) persistExecution(ctx context.Context, tree *merkle.Tree, treeCache *cache.Writer, numLeaves uint64) error {
	logging.FromContext(ctx).Info("persisting execution state", zap.Uint64("numLeaves", numLeaves), zap.String("round", r.ID))

	// Call GetReader() so that the cache would flush and validate structure.
	if _, err := treeCache.GetReader(); err != nil {
		return err
	}

	r.Execution.NumLeaves = numLeaves
	r.Execution.ParkedNodes = tree.GetParkedNodes()
	if err := r.saveState(); err != nil {
		return err
	}

	return nil
}

func (r *round) recoverExecution(ctx context.Context, end time.Time) error {
	logger := logging.FromContext(ctx).With(zap.String("round", r.ID))
	logger.With().Info("recovering execution", zap.Time("end", end))

	started := time.Now()

	if r.Execution.Members == nil || r.Execution.Statement == nil {
		logger.Debug("calculating members and statement")
		members, statement, err := r.calcMembersAndStatement()
		if err != nil {
			return fmt.Errorf("failed to calculate members and statement")
		}
		r.Execution.Members, r.Execution.Statement = members, statement
		if err := r.saveState(); err != nil {
			return err
		}
	}

	numLeaves, nip, err := prover.GenerateProofRecovery(
		ctx,
		r.datadir,
		hash.GenLabelHashFunc(r.Execution.Statement),
		hash.GenMerkleHashFunc(r.Execution.Statement),
		end,
		r.Execution.SecurityParam,
		r.Execution.NumLeaves,
		r.Execution.ParkedNodes,
		r.persistExecution,
	)
	if err != nil {
		return err
	}
	r.Execution.NumLeaves, r.Execution.NIP = numLeaves, nip
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
	if r.Execution.SecurityParam != state.Execution.SecurityParam {
		return errors.New("SecurityParam config mismatch")
	}
	r.roundState = state

	return nil
}

func (r *round) saveState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	return persist(filename, &r.roundState)
}

func (r *round) calcMembersAndStatement() ([][]byte, []byte, error) {
	mtree, err := merkle.NewTree()
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
		if err := os.RemoveAll(r.datadir); err != nil {
			return err
		}
	} else {
		return r.saveState()
	}

	return nil
}
