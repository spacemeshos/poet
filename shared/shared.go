package shared

import (
	"encoding/binary"
	"fmt"

	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/sha256-simd"
	"github.com/zeebo/blake3"
)

const (
	// T is the security param which determines the number of leaves
	// to be included in a non-interactive proof.
	T uint8 = 150
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

// MerkleProof is a non-interactive proof of inclusion in a Merkle tree.
// Scale encoding is implemented by hand to be able to limit [][]byte slices to a maximum size (inner and outer slices).
type MerkleProof struct {
	Root         []byte   `scale:"max=32"`
	ProvenLeaves [][]byte `scale:"max=150"`  // max. size is T (security param), and each element is exactly 32 bytes
	ProofNodes   [][]byte `scale:"max=5400"` // 36 nodes per leaf and each node is exactly 32 bytes
}

func (t *MerkleProof) EncodeScale(enc *scale.Encoder) (total int, err error) {
	{
		n, err := scale.EncodeByteSliceWithLimit(enc, t.Root, 32)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeLen(enc, uint32(len(t.ProvenLeaves)), 150)
		if err != nil {
			return total, fmt.Errorf("EncodeLen failed: %w", err)
		}
		total += n
		for _, byteSlice := range t.ProvenLeaves {
			n, err := scale.EncodeByteSliceWithLimit(enc, byteSlice, 32)
			if err != nil {
				return total, fmt.Errorf("EncodeByteSliceWithLimit failed: %w", err)
			}
			total += n
		}
	}
	{
		n, err := scale.EncodeLen(enc, uint32(len(t.ProofNodes)), 5400)
		if err != nil {
			return total, fmt.Errorf("EncodeLen failed: %w", err)
		}
		total += n
		for _, byteSlice := range t.ProofNodes {
			n, err := scale.EncodeByteSliceWithLimit(enc, byteSlice, 32)
			if err != nil {
				return total, fmt.Errorf("EncodeByteSliceWithLimit failed: %w", err)
			}
			total += n
		}
	}
	return total, nil
}

func (t *MerkleProof) DecodeScale(dec *scale.Decoder) (total int, err error) {
	{
		field, n, err := scale.DecodeByteSliceWithLimit(dec, 32)
		if err != nil {
			return total, err
		}
		total += n
		t.Root = field
	}
	{
		field, n, err := DecodeSliceOfByteSliceWithLimit(dec, 150, 32)
		if err != nil {
			return total, err
		}
		total += n
		t.ProvenLeaves = field
	}
	{
		field, n, err := DecodeSliceOfByteSliceWithLimit(dec, 5400, 32)
		if err != nil {
			return total, err
		}
		total += n
		t.ProofNodes = field
	}
	return total, nil
}

func DecodeSliceOfByteSliceWithLimit(d *scale.Decoder, outerLimit, innerLimit uint32) ([][]byte, int, error) {
	resultLen, total, err := scale.DecodeLen(d, outerLimit)
	if err != nil {
		return nil, 0, fmt.Errorf("DecodeLen failed: %w", err)
	}
	if resultLen == 0 {
		return nil, total, nil
	}
	result := make([][]byte, 0, resultLen)

	for i := uint32(0); i < resultLen; i++ {
		val, n, err := scale.DecodeByteSliceWithLimit(d, innerLimit)
		if err != nil {
			return nil, 0, fmt.Errorf("DecodeByteSlice failed: %w", err)
		}
		result = append(result, val)
		total += n
	}

	return result, total, nil
}

// HashMembershipTreeNode calculates internal node of
// the membership merkle tree.
func HashMembershipTreeNode(buf, lChild, rChild []byte) []byte {
	hasher := blake3.New()
	_, _ = hasher.Write([]byte{0x01})
	_, _ = hasher.Write(lChild)
	_, _ = hasher.Write(rChild)
	return hasher.Sum(buf)
}

// Non-Interactive proof of sequential work.
type NIP struct {
	MerkleProof
	Epoch  uint
	Leaves uint64
}
