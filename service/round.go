package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/smutil/log"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
)

type executionState struct {
	Epoch         uint32
	SecurityParam uint8
	Members       [][]byte
	Statement     []byte
	ParkedNodes   [][]byte
	NumLeaves     uint64
	NIP           *shared.MerkleProof
}

const roundStateFileBaseName = "state.bin"

type roundState struct {
	Opened           time.Time
	ExecutionStarted time.Time
	Execution        *executionState
}

func (r *roundState) isOpen() bool {
	return !r.Opened.IsZero() && r.ExecutionStarted.IsZero()
}

func (r *roundState) isExecuted() bool {
	return r.Execution.NIP != nil
}

type round struct {
	datadir string
	ID      string

	challengesDb *LevelDB
	execution    *executionState

	opened           time.Time
	executionStarted time.Time

	openedChan           chan struct{}
	executionStartedChan chan struct{}
	executionEndedChan   chan struct{}
	broadcastedChan      chan struct{}
	teardownChan         chan struct{}

	stateCache *roundState

	submitMtx sync.Mutex
}

func (r *round) Epoch() uint32 {
	return r.execution.Epoch
}

func newRound(ctx context.Context, datadir string, epoch uint32) *round {
	r := new(round)
	r.ID = strconv.FormatUint(uint64(epoch), 10)
	r.datadir = filepath.Join(datadir, r.ID)
	r.openedChan = make(chan struct{})
	r.executionStartedChan = make(chan struct{})
	r.executionEndedChan = make(chan struct{})
	r.broadcastedChan = make(chan struct{})
	r.teardownChan = make(chan struct{})

	wo := &opt.WriteOptions{Sync: true}
	r.challengesDb = NewLevelDbStore(filepath.Join(r.datadir, "challengesDb"), wo, nil) // This creates the datadir if it doesn't exist already.

	r.execution = new(executionState)
	r.execution.Epoch = epoch
	r.execution.SecurityParam = shared.T

	go func() {
		defer close(r.teardownChan)
		var cleanup bool
		select {
		case <-ctx.Done():
		case <-r.broadcastedChan:
			cleanup = true
		}

		if err := r.teardown(cleanup); err != nil {
			log.Error("Round %v tear down error: %v", r.ID, err)
			return
		}

		log.Info("Round %v torn down (cleanup %v)", r.ID, cleanup)
	}()

	return r
}

func (r *round) open() error {
	if r.stateCache != nil {
		r.opened = r.stateCache.Opened
	} else {
		r.opened = time.Now()
		if err := r.saveState(); err != nil {
			return err
		}
	}

	close(r.openedChan)

	return nil
}

func (r *round) waitTeardown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.teardownChan:
		return nil
	}
}

func (r *round) isOpen() bool {
	return !r.opened.IsZero() && r.executionStarted.IsZero()
}

func (r *round) submit(key, challenge []byte) error {
	if !r.isOpen() {
		return errors.New("round is not open")
	}

	r.submitMtx.Lock()
	defer r.submitMtx.Unlock()
	if has, err := r.challengesDb.Has(key); err != nil {
		return err
	} else if has {
		return fmt.Errorf("%w: key: %X", ErrChallengeAlreadySubmitted, key)
	}
	return r.challengesDb.Put(key, challenge)
}

func (r *round) numChallenges() int {
	iter := r.challengesDb.Iterator()
	defer iter.Release()

	var num int
	for iter.Next() {
		num++
	}

	return num
}

func (r *round) isEmpty() bool {
	iter := r.challengesDb.Iterator()
	defer iter.Release()
	return !iter.Next()
}

func (r *round) execute(ctx context.Context, end time.Time, minMemoryLayer uint) error {
	r.executionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	close(r.executionStartedChan)

	r.submitMtx.Lock()
	var err error
	r.execution.Members, r.execution.Statement, err = r.calcMembersAndStatement()
	if err != nil {
		return err
	}
	r.submitMtx.Unlock()

	if err := r.saveState(); err != nil {
		return err
	}

	r.execution.NumLeaves, r.execution.NIP, err = prover.GenerateProof(
		ctx,
		r.datadir,
		hash.GenLabelHashFunc(r.execution.Statement),
		hash.GenMerkleHashFunc(r.execution.Statement),
		end,
		r.execution.SecurityParam,
		minMemoryLayer,
		r.persistExecution,
	)
	if err != nil {
		return err
	}
	if err := r.saveState(); err != nil {
		return err
	}

	close(r.executionEndedChan)

	return nil
}

func (r *round) persistExecution(tree *merkle.Tree, treeCache *cache.Writer, numLeaves uint64) error {
	log.Info("Round %v: persisting execution state (done: %d)", r.ID, numLeaves)

	// Call GetReader() so that the cache would flush and validate structure.
	if _, err := treeCache.GetReader(); err != nil {
		return err
	}

	r.execution.NumLeaves = numLeaves
	r.execution.ParkedNodes = tree.GetParkedNodes()
	if err := r.saveState(); err != nil {
		return err
	}

	return nil
}

func (r *round) recoverExecution(ctx context.Context, state *executionState, end time.Time) error {
	r.executionStarted = r.stateCache.ExecutionStarted
	close(r.executionStartedChan)

	if state.Members != nil && state.Statement != nil {
		r.execution.Members = state.Members
		r.execution.Statement = state.Statement
	} else {
		var err error
		r.execution.Members, r.execution.Statement, err = r.calcMembersAndStatement()
		if err != nil {
			return err
		}
		if err := r.saveState(); err != nil {
			return err
		}
	}

	var err error
	r.execution.NumLeaves, r.execution.NIP, err = prover.GenerateProofRecovery(
		ctx,
		r.datadir,
		hash.GenLabelHashFunc(state.Statement),
		hash.GenMerkleHashFunc(state.Statement),
		end,
		state.SecurityParam,
		state.NumLeaves,
		state.ParkedNodes,
		r.persistExecution,
	)
	if err != nil {
		return err
	}
	if err := r.saveState(); err != nil {
		return err
	}

	close(r.executionEndedChan)

	return nil
}

func (r *round) proof(wait bool) (*PoetProof, error) {
	if wait {
		<-r.executionEndedChan
	} else {
		select {
		case <-r.executionEndedChan:
		default:
			select {
			case <-r.executionStartedChan:
				return nil, errors.New("round is executing")
			default:
				select {
				case <-r.openedChan:
					return nil, errors.New("round is open")
				default:
					return nil, errors.New("round wasn't open")
				}
			}
		}
	}

	return &PoetProof{
		N:         uint(r.execution.NumLeaves),
		Statement: r.execution.Statement,
		Proof:     r.execution.NIP,
	}, nil
}

func (r *round) broadcasted() {
	close(r.broadcastedChan)
}

func (r *round) state() (*roundState, error) {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	s := &roundState{}

	if err := load(filename, s); err != nil {
		return nil, err
	}
	if r.execution.SecurityParam != s.Execution.SecurityParam {
		return nil, errors.New("SecurityParam config mismatch")
	}
	r.stateCache = s

	return s, nil
}

func (r *round) saveState() error {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	v := &roundState{
		Opened:           r.opened,
		ExecutionStarted: r.executionStarted,
		Execution:        r.execution,
	}
	return persist(filename, v)
}

func (r *round) calcMembersAndStatement() ([][]byte, []byte, error) {
	mtree, err := merkle.NewTree()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize merkle tree: %v", err)
	}

	members := make([][]byte, 0)
	iter := r.challengesDb.Iterator()
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
	}

	return nil
}
