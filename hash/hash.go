package hash

import (
	"github.com/spacemeshos/sha256-simd"
)

// LabelHashNestingDepth is the number of recursive hashes per label.
const LabelHashNestingDepth = 100

// GenMerkleHashFunc generates Merkle hash functions salted with a challenge. The challenge is prepended to the
// concatenation of the left- and right-child in the tree and the result is hashed using Sha256.
//
// ⚠️ The resulting function is NOT thread-safe, however different generated instances are independent.
// The code is optimized for performance and memory allocations.
func GenMerkleHashFunc(challenge []byte) func(lChild, rChild []byte) []byte {
	var buffer []byte
	return func(lChild, rChild []byte) []byte {
		size := len(challenge) + len(lChild) + len(rChild)
		if len(buffer) < size {
			buffer = make([]byte, size)
		}
		copy(buffer, challenge)
		copy(buffer[len(challenge):], lChild)
		copy(buffer[len(challenge)+len(lChild):], rChild)

		result := sha256.Sum256(buffer[:size])
		return result[:]
	}
}

func NewGenMerkleHashFunc(challenge []byte) func(lChild, rChild []byte) []byte {
	// De-virtualize the call to sha256 hasher
	switch h := sha256.New().(type) {
	case *sha256.Digest:
		return func(lChild, rChild []byte) []byte {
			h.Reset()
			h.Write(challenge)
			h.Write(lChild)
			h.Write(rChild)
			hash := h.CheckSum()
			return hash[:]
		}
	default:
		return func(lChild, rChild []byte) []byte {
			h.Reset()
			h.Write(challenge)
			h.Write(lChild)
			h.Write(rChild)
			return h.Sum(nil) // TODO reuse buffer (impossible right not because it is persisted in merkle-tree)
		}
	}
}

// GenLabelHashFunc generates hash functions for computing labels. The challenge is prepended to the data and the result
// is hashed using Sha256. TODO: use nested hashes based on a difficulty param.
func GenLabelHashFunc(challenge []byte) func(data []byte) []byte {
	return func(data []byte) []byte {
		message := append(challenge, data...)
		var res [32]byte
		for i := 0; i < LabelHashNestingDepth; i++ {
			res = sha256.Sum256(message)
			message = res[:]
		}
		return message
	}
}

func NewGenLabelHashFunc(challenge []byte) func(data []byte) []byte {
	var hashBuf [32]byte
	// De-virtualize the call to sha256 hasher
	switch h := sha256.New().(type) {
	case *sha256.Digest:
		return func(data []byte) []byte {
			h.Reset()
			h.Write(challenge)
			h.Write(data)
			hashBuf = h.CheckSum()

			for i := 1; i < LabelHashNestingDepth; i++ {
				h.Reset()
				h.Write(hashBuf[:])
				hashBuf = h.CheckSum()
			}
			return hashBuf[:]
		}
	default:
		return func(data []byte) []byte {
			h.Reset()
			h.Write(challenge)
			h.Write(data)
			hash := h.Sum(hashBuf[:0])

			for i := 1; i < LabelHashNestingDepth; i++ {
				h.Reset()
				h.Write(hash[:])
				hash = h.Sum(hash[:0])
			}
			return hash[:]
		}
	}
}
