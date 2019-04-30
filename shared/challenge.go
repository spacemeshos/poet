package shared

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/sha256-simd"
)

type LabelHashFunc func(data []byte) []byte

type Sha256Challenge []byte

func (c Sha256Challenge) MerkleHashFunc() merkle.HashFunc {
	return func(lChild, rChild []byte) []byte {
		children := append(lChild, rChild...)
		result := sha256.Sum256(append(c, children...))
		return result[:]
	}
}

func (c Sha256Challenge) LabelHashFunc() LabelHashFunc {
	return func(data []byte) []byte {
		result := sha256.Sum256(append(c, data...))
		return result[:]
	}
}

