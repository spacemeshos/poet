package shared

import (
	"encoding/binary"
	"os"

	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/sha256-simd"
)

//go:generate scalegen -types MerkleProof,ProofMessage,Proof,Member,Leaf,Node

const (
	// T is the security param which determines the number of leaves
	// to be included in a non-interactive proof.
	T uint8 = 150

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0o600)
)

type LabelHash func(data []byte) []byte

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
	digest := make([]byte, sha256.Size)
	hasher := sha256.New()
	for i := uint32(0); len(ret) < int(securityParam); i++ {
		binary.BigEndian.PutUint32(ib, i)
		hasher.Reset()
		hasher.Write(challenge)
		hasher.Write(ib)
		digest = hasher.Sum(digest[:0])
		id := binary.BigEndian.Uint64(digest[:8]) % spaceSize
		ret[id] = true
	}
	return ret
}

// MakeLabelFunc returns a function which generates a PoET DAG label by concatenating a representation
// of the labelID with the list of left siblings and then hashing the result using the provided hash function.
//
// ⚠️ The resulting function is NOT thread-safe, however different generated instances are independent.
// The code is optimized for performance and memory allocations.
func MakeLabelFunc() func(hash LabelHash, labelID uint64, leftSiblings [][]byte) []byte {
	var buffer []byte
	return func(hash LabelHash, labelID uint64, leftSiblings [][]byte) []byte {
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

type Member struct {
	Challenge []byte `scale:"max=32"`
}

type Proof struct {
	// The actual proof.
	MerkleProof

	// Members is the ordered list of miners challenges which are included
	// in the proof (by using the list hash digest as the proof generation input (the statement)).
	Members []Member `scale:"max=150"`

	// NumLeaves is the width of the proof-generation tree.
	NumLeaves uint64
}

type ProofMessage struct {
	Proof
	ServicePubKey []byte `scale:"max=32"`
	RoundID       string `scale:"max=32"`
}

type Leaf struct {
	Value []byte `scale:"max=32"`
}

type Node struct {
	Value []byte `scale:"max=32"`
}

type MerkleProof struct {
	Root         []byte `scale:"max=32"`
	ProvenLeaves []Leaf `scale:"max=150"`  // the max. size of this slice is T (security param)
	ProofNodes   []Node `scale:"max=5400"` // 36 nodes per leaf
}
