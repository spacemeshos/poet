package types

import (
	"context"
	"encoding/hex"

	"github.com/spacemeshos/poet/shared"
	"go.uber.org/zap/zapcore"
)

//go:generate scalegen -types NIPostChallenge

type ATX struct {
	NodeID     []byte
	Sequence   uint64
	PubLayerID shared.LayerID
}

func (a *ATX) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if a == nil {
		return nil
	}
	encoder.AddString("node_id", hex.EncodeToString(a.NodeID))
	encoder.AddUint32("pub_layer_id", uint32(a.PubLayerID))
	encoder.AddUint64("sequence", a.Sequence)
	return nil
}

//go:generate mockgen -package mocks -destination mocks/atx_provider.go . AtxProvider
type AtxProvider interface {
	Get(context.Context, shared.ATXID) (*ATX, error)
}

type PostConfig struct {
	MinNumUnits   uint32
	MaxNumUnits   uint32
	BitsPerLabel  uint8
	LabelsPerUnit uint64
	K1            uint32
	K2            uint32
}

type NIPostChallenge struct {
	// Sequence number counts the number of ancestors of the ATX. It sequentially increases for each ATX in the chain.
	// Two ATXs with the same sequence number from the same miner can be used as the proof of malfeasance against that miner.
	Sequence       uint64
	PrevATXID      shared.ATXID
	PubLayerID     uint32
	PositioningATX shared.ATXID

	// CommitmentATX is the ATX used in the commitment for initializing the PoST of the node.
	CommitmentATX      *shared.ATXID
	InitialPostIndices []byte
}
