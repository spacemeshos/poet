package shared

import (
	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/post/shared"
)

//go:generate scalegen -types InitialPost,Challenge

type ATXID [32]byte

func (t *ATXID) EncodeScale(e *scale.Encoder) (int, error) {
	return scale.EncodeByteArray(e, t[:])
}

// DecodeScale implements scale codec interface.
func (t *ATXID) DecodeScale(d *scale.Decoder) (int, error) {
	return scale.DecodeByteArray(d, t[:])
}

type InitialPost struct {
	Proof           shared.Proof
	Metadata        shared.ProofMetadata
	CommitmentAtxId ATXID
}

type LayerID uint32

type Challenge struct {
	PositioningAtxId ATXID
	PubLayerId       LayerID
	// only one of InitialPost, previousATX is valid at the same time
	InitialPost   *InitialPost
	PreviousATXId *ATXID
}
