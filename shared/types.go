package shared

import "github.com/spacemeshos/post/shared"

type ATXID []byte

//go:generate scalegen -types InitialPost,Challenge

type InitialPost struct {
	Proof    shared.Proof
	Metadata shared.ProofMetadata
}

type Challenge struct {
	NodeID           []byte
	PositioningAtxId []byte
	PubLayerId       []byte
	// only one of InitialPost, previousATX is valid at the same time
	InitialPost   *InitialPost
	PreviousATXId []byte
}
