package shared

import (
	"encoding/hex"

	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"go.uber.org/zap/zapcore"
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

func (t *ATXID) Field() log.Field {
	if t == nil {
		return log.String("atx_id", "<nil>")
	}
	return log.String("atx_id", hex.EncodeToString(t[:]))
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

func (p *InitialPost) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if p == nil {
		return nil
	}
	encoder.AddUint32("proof-nonce", p.Proof.Nonce)
	encoder.AddBinary("proof-indicies", p.Proof.Indices)
	encoder.AddString("commitment_atx_id", p.CommitmentAtxId.Field().String)
	return nil
}

func (c *Challenge) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if c == nil {
		return nil
	}
	encoder.AddString("positioning_atx_id", c.PositioningAtxId.Field().String)
	encoder.AddUint32("pub_layer_id", uint32(c.PubLayerId))
	encoder.AddObject("initial_post", c.InitialPost)

	encoder.AddString("previous_atx_id", c.PreviousATXId.Field().String)
	return nil
}
