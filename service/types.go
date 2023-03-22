package service

import "github.com/spacemeshos/poet/shared"

type Proof struct {
	// The actual proof.
	shared.MerkleProof

	// Members is the ordered list of miners challenges which are included
	// in the proof (by using the list hash digest as the proof generation input (the statement)).
	Members [][]byte

	// NumLeaves is the width of the proof-generation tree.
	NumLeaves uint64
}

type ProofMessage struct {
	Proof
	ServicePubKey []byte
	RoundID       string
}
