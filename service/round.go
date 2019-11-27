package service

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type executionState struct {
	NumLeaves     uint64
	SecurityParam uint8
	Members       [][]byte
	Statement     []byte
	ParkedNodes   [][]byte
	NextLeafId    uint64
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

type round struct {
	cfg     *Config
	datadir string
	Id      string

	challengesDb *LevelDB
	execution    *executionState

	opened           time.Time
	executionStarted time.Time

	openedChan           chan struct{}
	executionStartedChan chan struct{}
	executionEndedChan   chan struct{}

	sig       *signal.Signal
	submitMtx sync.Mutex
}

func newRound(sig *signal.Signal, cfg *Config, datadir string, id string) *round {
	r := new(round)
	r.cfg = cfg
	r.datadir = datadir
	r.Id = id
	r.openedChan = make(chan struct{})
	r.executionStartedChan = make(chan struct{})
	r.executionEndedChan = make(chan struct{})
	r.sig = sig

	dbPath := filepath.Join(datadir, "challengesDb")
	wo := &opt.WriteOptions{Sync: true}
	r.challengesDb = NewLevelDbStore(dbPath, wo, nil) // This creates the datadir if it doesn't exist already.

	r.execution = new(executionState)
	r.execution.NumLeaves = uint64(1) << r.cfg.N // TODO(noamnelke): configure tick count instead of height
	r.execution.SecurityParam = shared.T

	go func() {
		var cleanup bool
		select {
		case <-sig.ShutdownRequestedChan:
		case <-r.executionEndedChan:
			cleanup = true
		}

		if err := r.teardown(cleanup); err != nil {
			log.Error("Round %v tear down error: %v", r.Id, err)
			return
		}

		log.Info("Round %v teared down", r.Id)
	}()

	return r
}

func (r *round) open() error {
	r.opened = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

	close(r.openedChan)

	return nil
}

func (r *round) isOpen() bool {
	return !r.opened.IsZero() && r.executionStarted.IsZero()
}

func (r *round) submit(challenge []byte) error {
	if !r.isOpen() {
		return errors.New("round is not open")
	}

	r.submitMtx.Lock()
	err := r.challengesDb.Put(challenge, nil)
	r.submitMtx.Unlock()

	return err
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

func (r *round) execute() error {
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

	minMemoryLayer := int(r.cfg.N - r.cfg.MemoryLayers)
	if minMemoryLayer < prover.LowestMerkleMinMemoryLayer {
		minMemoryLayer = prover.LowestMerkleMinMemoryLayer
	}

	r.execution.NIP, err = prover.GenerateProof(
		r.sig,
		r.datadir,
		hash.GenLabelHashFunc(r.execution.Statement),
		hash.GenMerkleHashFunc(r.execution.Statement),
		r.execution.NumLeaves,
		r.execution.SecurityParam,
		uint(minMemoryLayer),
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

func (r *round) persistExecution(tree *merkle.Tree, treeCache *cache.Writer, nextLeafId uint64) error {
	log.Info("Round %v: persisting execution state (done: %d, total: %d)", r.Id, nextLeafId, r.execution.NumLeaves)

	// Call GetReader() so that the cache would flush and validate structure.
	if _, err := treeCache.GetReader(); err != nil {
		return err
	}

	r.execution.NextLeafId = nextLeafId
	r.execution.ParkedNodes = tree.GetParkedNodes()
	if err := r.saveState(); err != nil {
		return err
	}

	return nil
}

func (r *round) recoverExecution(state *executionState) error {
	r.executionStarted = time.Now()
	if err := r.saveState(); err != nil {
		return err
	}

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
	r.execution.NIP, err = prover.GenerateProofRecovery(
		r.sig,
		r.datadir,
		hash.GenLabelHashFunc(state.Statement),
		hash.GenMerkleHashFunc(state.Statement),
		state.NumLeaves,
		state.SecurityParam,
		state.NextLeafId,
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
		N:         r.cfg.N,
		Statement: r.execution.Statement,
		Proof:     r.execution.NIP,
	}, nil
}

func (r *round) state() (*roundState, error) {
	filename := filepath.Join(r.datadir, roundStateFileBaseName)
	s := &roundState{}

	if err := load(filename, s); err != nil {
		return nil, err
	}

	if r.execution.NumLeaves != s.Execution.NumLeaves {
		return nil, errors.New("NumLeaves config mismatch")
	}
	if r.execution.SecurityParam != s.Execution.SecurityParam {
		return nil, errors.New("SecurityParam config mismatch")
	}

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
		key := iter.Key()
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)

		members = append(members, keyCopy)
		if err := mtree.AddLeaf(keyCopy); err != nil {
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
