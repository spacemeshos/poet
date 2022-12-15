package shared

import (
	"encoding/binary"
	"os"

	"github.com/minio/sha256-simd"
	"github.com/spacemeshos/merkle-tree"
)

//go:generate scalegen -types MerkleProof,ProofMessage,Proof

const (
	// T is the security param which determines the number of leaves
	// to be included in a non-interactive proof.
	T uint8 = 150

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0o600)
)

// FiatShamir generates a set of indices to include in a non-interactive proof.
func FiatShamir(challenge []byte, spaceSize uint64, securityParam uint8) map[uint64]bool {
	ret := make(map[uint64]bool, securityParam)
	if uint64(securityParam) > spaceSize {
		for i := uint64(0); i < spaceSize; i++ {
			ret[i] = true
		}
		return ret
	}

	ib := make([]byte, 4)
	for i := uint32(0); len(ret) < int(securityParam); i++ {
		binary.BigEndian.PutUint32(ib, i)
		result := sha256.Sum256(append(challenge, ib...))
		id := binary.BigEndian.Uint64(result[:8]) % spaceSize
		ret[id] = true
	}
	return ret
}

// MakeLabelFunc returns a function which generates a PoET DAG label by concatenating a representation
// of the labelID with the list of left siblings and then hashing the result using the provided hash function.
//
// ⚠️ The resulting function is NOT thread-safe, however different generated instances are independent.
// The code is optimized for performance and memory allocations.
func MakeLabelFunc() func(hash func(data []byte) []byte, labelID uint64, leftSiblings [][]byte) []byte {
	var buffer []byte
	return func(hash func(data []byte) []byte, labelID uint64, leftSiblings [][]byte) []byte {
		// Calculate the buffer required size.
		// 8 is for the size of labelID.
		// leftSiblings slice might contain nil values, so the result size is inflated and used as an upper bound.
		size := 8 + len(leftSiblings)*merkle.NodeSize

		if len(buffer) < size {
			buffer = make([]byte, size)
		}

		binary.BigEndian.PutUint64(buffer, labelID)
		offset := 8

		for _, sibling := range leftSiblings {
			copied := copy(buffer[offset:], sibling)
			offset += copied
		}
		sum := hash(buffer[:offset])
		return sum
	}
}

type Proof struct {
	// The actual proof.
	MerkleProof

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

type MerkleProof struct {
	Root         []byte
	ProvenLeaves [][]byte
	ProofNodes   [][]byte
}
