package shared

import "github.com/spacemeshos/post/shared"

type ATXID []byte

//go:generate scalegen -types InitialPost,Challenge

type InitialPost struct {
	Proof    shared.Proof
	Metadata shared.ProofMetadata
}

type LayerID uint32

type Challenge struct {
	PositioningAtxId ATXID
	PubLayerId       LayerID
	// only one of InitialPost, previousATX is valid at the same time
	InitialPost   *InitialPost
	PreviousATXId ATXID
}
