package service

import (
	"bytes"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
)

type round struct {
	cfg *Config
	id  uint

	commitments [][]byte
	tree        *merkle.Tree
	treeRoot    []byte
	phi         []byte

	closedChan   chan struct{}
	executedChan chan struct{}
}

func newRound(cfg *Config, id uint) *round {
	r := new(round)
	r.cfg = cfg
	r.id = id
	r.closedChan = make(chan struct{})
	r.executedChan = make(chan struct{})

	return r
}

func (r *round) submitCommitment(c []byte) error {
	r.commitments = append(r.commitments, c)

	return nil
}

func (r *round) close() error {
	r.tree = merkle.NewTree(merkle.GetSha256Parent)
	for _, c := range r.commitments {
		err := r.tree.AddLeaf(c)
		if err != nil {
			return err
		}
	}

	root, err := r.tree.Root()
	if err != nil {
		return nil
	}
	r.treeRoot = root

	close(r.closedChan)
	return nil
}

func (r *round) execute() (shared.Label, error) {
	prover, err := internal.NewProver(r.treeRoot, r.cfg.N, shared.NewHashFunc(r.treeRoot))
	if err != nil {
		return shared.Label{}, err
	}

	phi, err := prover.ComputeDag()
	if err != nil {
		return shared.Label{}, err
	}
	r.phi = phi

	close(r.executedChan)
	return phi, nil
}

func (r *round) membershipProof(c []byte) ([][]byte, error) {
	// temporary inefficient implementation
	var ci uint64
	for i, commitment := range r.commitments {
		if bytes.Equal(c, commitment) {
			ci = uint64(i)
			break
		}
	}

	t := merkle.NewProvingTree(merkle.GetSha256Parent, []uint64{ci})
	for _, c := range r.commitments {
		err := t.AddLeaf(c)
		if err != nil {
			return nil, err
		}
	}

	treeRoot, err := t.Root()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(treeRoot, r.treeRoot) {
		return nil, fmt.Errorf("incorrect tree root: %v", treeRoot)
	}

	proof, err := t.Proof()
	if err != nil {
		return nil, err
	}

	return proof, nil
}
