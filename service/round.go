package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"time"
)

type round struct {
	cfg *Config

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

func (r *round) membershipProof(challenge []byte, wait bool) (*MembershipProof, error) {
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
	for i, ch := range r.challenges {
		if bytes.Equal(challenge, ch) {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, errors.New("challenge not found")
	}

	var leavesToProve = make(map[uint64]bool)
	leavesToProve[uint64(index)] = true

	t, err := merkle.NewProvingTree(leavesToProve)
	if err != nil {
		return nil, fmt.Errorf("could not initialize merkle tree: %v", err)
	}
	for _, c := range r.challenges {
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

	return &MembershipProof{
		Index: index,
		Root:  r.merkleRoot,
		Proof: proof,
	}, nil

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
