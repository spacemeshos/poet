package hash

import (
	"github.com/minio/sha256-simd"
	"github.com/spacemeshos/merkle-tree"
)

// LabelHashNestingDepth is the number of recursive hashes per label.
const LabelHashNestingDepth = 100

// GenMerkleHashFunc generates Merkle hash functions salted with a challenge. The challenge is prepended to the
// concatenation of the left- and right-child in the tree and the result is hashed using Sha256.
//
// ⚠️ The resulting function is NOT thread-safe, however different generated instances are independent.
// The code is optimized for performance and memory allocations.
func GenMerkleHashFunc(challenge []byte) merkle.HashFunc {
	h := sha256.New()
	return func(buf, lChild, rChild []byte) []byte {
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(lChild)
		_, _ = h.Write(rChild)
		return h.Sum(buf)
	}
}

// GenLabelHashFunc generates hash functions for computing labels. The challenge is prepended to the data and the result
// is hashed using Sha256. TODO: use nested hashes based on a difficulty param.
func GenLabelHashFunc(challenge []byte) func(data []byte) []byte {
	var hashBuf [sha256.Size]byte
	h := sha256.New()
	return func(data []byte) []byte {
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(data)
		hash := h.Sum(hashBuf[:0])

		for i := 1; i < LabelHashNestingDepth; i++ {
			h.Reset()
			_, _ = h.Write(hash)
			hash = h.Sum(hash[:0])
		}
		return hash[:]
	}
}
