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
			return h.Sum(nil)
		}
	}
}

// GenLabelHashFunc generates hash functions for computing labels. The challenge is prepended to the data and the result
// is hashed using Sha256. TODO: use nested hashes based on a difficulty param.
func GenLabelHashFunc(challenge []byte) func(data []byte) []byte {
	var hashBuf [32]byte
	// Try to de-virtualize the call to sha256 hasher
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
