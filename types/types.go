package types

import (
	"context"

	"github.com/spacemeshos/poet/shared"
)

type ATX struct {
	NodeID     []byte
	Sequence   uint64
	PubLayerID shared.LayerID
}

//go:generate mockgen -destination mock_types/atx_provider.go . AtxProvider
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
