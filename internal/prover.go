package internal

import (
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
)

// A simple poet prover
// For now - Holds a DAG in RAM - useful for generating proofs for small values of n

type IProver interface {

}

type SMProver struct {
	x []byte   // commitment
	n uint     // n param 1 <= n <= 63
	h HashFunc // Hx()
	m map[string]shared.Label // label storage
}

// Create a new verifier for commitment X and param n
func NewProver(x []byte, n uint) (IProver, error) {

	if n < 1 || n > 63 {
		return nil, errors.New("n must be in range [1, 63]")
	}

	res := &SMProver{
		x: x,
		n: n,
		h: shared.NewHashFunc(x),
		m: make(map[string]shared.Label),
	}

	return res, nil
}


/*
Compute the labels of the left subtree (tree with root l0)
Keep the label of l0 in memory and discard all other computed labels from memory
Compute the labels of the right subtree (tree with root l1) - using l0
Once l1 is computed, discard all other computed labels from memory and keep l1
Compute the root label le = Hx("", l0, l1)
When a label value is computed by the algorithm, store it in persistent storage if the label's height <= m.
Note that this works because only l0 is needed for computing labels in the tree rooted in l1. All of the additional edges to nodes in the tree rooted at l1 start at l0.
Note that the reference Python code does not construct the DAG in this manner and keeps the whole DAG in memory. Please use the Python code as an example for simpler constructions such as binary strings, open and verify.
*/

func BuildDag(height uint) (shared.Label) {
	return shared.Label{}
}