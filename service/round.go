package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	prover "github.com/spacemeshos/poet-ref/prover"
	"github.com/spacemeshos/poet-ref/shared"
	"time"
)

type round struct {
	cfg *Config

	Id           int
	opened       time.Time
	executeStart time.Time
	executeEnd   time.Time

	commits    [][]byte
	merkleTree *merkle.Tree
	merkleRoot []byte
	nip        *shared.Proof

	closedChan   chan struct{}
	executedChan chan struct{}
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

func (r *round) submit(data []byte) error {
	// TODO(moshababo): check for duplications?
	r.commits = append(r.commits, data)

	return nil
}

func (r *round) close() error {
	r.merkleTree = merkle.NewTree()
	for _, c := range r.commits {
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
	// TODO(moshababo): use the config hash function
	prover, err := prover.New(r.merkleRoot, r.cfg.N, shared.NewHashFunc(r.merkleRoot))
	if err != nil {
		return err
	}

	r.executeStart = time.Now()
	_, err = prover.ComputeDag()
	if err != nil {
		return err
	}
	nip, err := prover.GetNonInteractiveProof()
	if err != nil {
		return err
	}

	prover.DeleteStore()

	r.executeEnd = time.Now()
	r.nip = &nip
	close(r.executedChan)
	return nil

}

func (r *round) membershipProof(c []byte, wait bool) (*shared.MembershipProof, error) {
	if wait {
		<-r.closedChan
	} else {
		select {
		case <-r.closedChan:
		default:
			return nil, errors.New("round is open")
		}
	}

	// TODO(moshababo): change this temp inefficient implementation
	index := -1
	for i, commit := range r.commits {
		if bytes.Equal(c, commit) {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, errors.New("commit not found")
	}

	var leavesToProve = make(map[uint64]bool)
	leavesToProve[uint64(index)] = true

	t := merkle.NewProvingTree(leavesToProve)
	for _, c := range r.commits {
		err := t.AddLeaf(c)
		if err != nil {
			return nil, err
		}
	}

	merkleRoot := t.Root()
	if !bytes.Equal(t.Root(), r.merkleRoot) {
		return nil, fmt.Errorf("incorrect merkleTree root, expected: %x, found: %x", r.merkleRoot, merkleRoot)
	}

	proof := t.Proof()

	return &shared.MembershipProof{
		Index: index,
		Root:  r.merkleRoot,
		Proof: proof,
	}, nil

}

func (r *round) proof(wait bool) (*shared.Proof, error) {
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

	return r.nip, nil
}
