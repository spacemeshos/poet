package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"time"
)

type round struct {
	cfg          *Config
	id           int
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
	r.id = id
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
	r.merkleTree = merkle.NewTree(merkle.GetSha256Parent)
	for _, c := range r.commits {
		err := r.merkleTree.AddLeaf(c)
		if err != nil {
			return err
		}
	}

	root, err := r.merkleTree.Root()
	if err != nil {
		return err
	}

	r.merkleRoot = root

	close(r.closedChan)
	return nil
}

func (r *round) execute() error {
	// TODO(moshababo): use the config hash function
	prover, err := internal.NewProver(r.merkleRoot, r.cfg.N, shared.NewHashFunc(r.merkleRoot))
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

	r.executeEnd = time.Now()
	r.nip = &nip
	close(r.executedChan)
	return nil
}

func (r *round) membershipProof(c []byte) ([][]byte, error) {
	// TODO(moshababo): change this temp inefficient implementation
	ci := -1
	for i, commit := range r.commits {
		if bytes.Equal(c, commit) {
			ci = i
			break
		}
	}

	if ci == -1 {
		return nil, errors.New("commit not found")
	}

	t := merkle.NewProvingTree(merkle.GetSha256Parent, []uint64{uint64(ci)})
	for _, c := range r.commits {
		err := t.AddLeaf(c)
		if err != nil {
			return nil, err
		}
	}

	merkleRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(merkleRoot, r.merkleRoot) {
		return nil, fmt.Errorf("incorrect merkleTree root: %v", merkleRoot)
	}

	proof, err := t.Proof()
	if err != nil {
		return nil, err
	}

	return proof, nil
}
