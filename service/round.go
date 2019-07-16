package service

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"sync"
	"time"
)

type round struct {
	cfg    *Config

	Id           int
	opened       time.Time
	executeStart time.Time
	executeEnd   time.Time

	challenges [][]byte
	merkleTree *merkle.Tree
	merkleRoot []byte
	nip        *shared.MerkleProof

	closedChan   chan struct{}
	executedChan chan struct{}

	sync.Mutex
}

func newRound(cfg *Config, id int) *round {
	r := new(round)
	r.cfg = cfg
	r.Id = id
	r.opened = time.Now()
	r.closedChan = make(chan struct{})
	r.executedChan = make(chan struct{})

	return r
}

func (r *round) submit(challenge []byte) error {
	r.Lock()
	defer r.Unlock()

	// TODO(moshababo): check for duplications?
	r.challenges = append(r.challenges, challenge)

	return nil
}

func (r *round) close() error {
	var err error
	r.merkleTree, err = merkle.NewTree()
	if err != nil {
		return fmt.Errorf("could not initialize merkle tree: %v", err)
	}
	for _, c := range r.challenges {
		err := r.merkleTree.AddLeaf(c)
		if err != nil {
			return err
		}
	}

	r.merkleRoot = r.merkleTree.Root()

	close(r.closedChan)
	return nil
}

func (r *round) execute() error {
	challenge := r.merkleRoot
	leafCount := uint64(1) << r.cfg.N // TODO(noamnelke): configure tick count instead of height
	securityParam := shared.T

	r.executeStart = time.Now()
	nip, err := prover.GetProof(hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafCount, securityParam)
	if err != nil {
		return err
	}

	r.executeEnd = time.Now()
	r.nip = &nip
	close(r.executedChan)
	return nil

}

func (r *round) proof(wait bool) (*PoetProof, error) {
	if wait {
		<-r.executedChan
	} else {
		select {
		case <-r.executedChan:
		default:
			select {
			case <-r.closedChan:
				return nil, errors.New("round is executing")
			default:
				return nil, errors.New("round is open")
			}
		}
	}

	return &PoetProof{
		N:          r.cfg.N,
		Commitment: r.merkleRoot,
		Proof:      r.nip,
	}, nil
}
