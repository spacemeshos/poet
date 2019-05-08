package shared

import (
	"encoding/binary"
	"github.com/spacemeshos/sha256-simd"
)

const T uint8 = 150 // security param

// FiatShamir generates a set of indices to include in a non-interactive proof.
func FiatShamir(challenge []byte, spaceSize uint64, indexCount uint8) map[uint64]bool {
	if uint64(indexCount) > spaceSize {
		indexCount = uint8(spaceSize)
	}
	ret := make(map[uint64]bool, indexCount)
	for i := uint8(0); len(ret) < int(indexCount); i++ {
		result := sha256.Sum256(append(challenge, i))
		id := binary.BigEndian.Uint64(result[:8]) % spaceSize
		ret[id] = true
	}
	return ret
}

// MakeLabel generates a PoET DAG label by concatenating a representation of the labelID with the list of left siblings
// and then hashing the result using the provided hash function.
func MakeLabel(hash func(data []byte) []byte, labelID uint64, leftSiblings [][]byte) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, labelID)
	for _, sibling := range leftSiblings {
		data = append(data, sibling...)
	}
	sum := hash(data)
	//fmt.Printf("label %2d: %x | data: %x\n", labelID, sum, data)
	return sum
}

type MerkleProof struct {
	Root         []byte
	ProvenLeaves [][]byte
	ProofNodes   [][]byte
}
