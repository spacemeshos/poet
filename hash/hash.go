package hash

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/sha256-simd"

	"github.com/spacemeshos/poet/shared"
)

// LabelHashNestingDepth is the number of recursive hashes per label.
const LabelHashNestingDepth = 100

// GenMerkleHashFunc generates Merkle hash functions salted with a challenge. The challenge is prepended to the
// concatenation of the left- and right-child in the tree and the result is hashed using Sha256.
//
// ⚠️ The resulting function is NOT thread-safe, however different generated instances are independent.
func GenMerkleHashFunc(challenge []byte) merkle.HashFunc {
	// De-virtualize the call to sha256 hasher
	switch sha256.New().(type) {
	case *sha256.Digest:
		return genMerkleHashFuncMinio(challenge)
	default:
		return genMerkleHashFuncGeneric(challenge)
	}
}

// generate merkle HashFunc assuming SHA extensions / ARM sha2
// is supported. Panics otherwise.
func genMerkleHashFuncMinio(challenge []byte) merkle.HashFunc {
	h := sha256.New().(*sha256.Digest)
	return func(buf, lChild, rChild []byte) []byte {
		if cap(buf) < sha256.Size {
			buf = make([]byte, sha256.Size)
		} else {
			buf = buf[:sha256.Size]
		}
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(lChild)
		_, _ = h.Write(rChild)
		h.CheckSumInto((*[sha256.Size]byte)(buf))
		return buf
	}
}

func genMerkleHashFuncGeneric(challenge []byte) merkle.HashFunc {
	h := sha256.New()
	return func(buf, lChild, rChild []byte) []byte {
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(lChild)
		_, _ = h.Write(rChild)
		return h.Sum(buf[:0])
	}
}

// GenLabelHashFunc generates hash functions for computing labels. The challenge is prepended to the data and the result
// is hashed using Sha256. TODO: use nested hashes based on a difficulty param.
func GenLabelHashFunc(challenge []byte) shared.LabelHash {
	// Try to de-virtualize the call to sha256 hasher
	switch sha256.New().(type) {
	case *sha256.Digest:
		return genLabelHashFuncMinio(challenge)
	default:
		return genLabelHashFuncGeneric(challenge)
	}
}

// generate label HashFunc assuming SHA extensions / ARM sha2
// is supported. Panics otherwise.
func genLabelHashFuncMinio(challenge []byte) shared.LabelHash {
	var hashBuf [sha256.Size]byte
	h := sha256.New().(*sha256.Digest)
	return func(data []byte) []byte {
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(data)
		h.CheckSumInto(&hashBuf)
		for i := 1; i < LabelHashNestingDepth; i++ {
			h.Reset()
			_, _ = h.Write(hashBuf[:])
			h.CheckSumInto(&hashBuf)
		}
		return hashBuf[:]
	}
}

func genLabelHashFuncGeneric(challenge []byte) shared.LabelHash {
	var hashBuf [sha256.Size]byte
	h := sha256.New()
	return func(data []byte) []byte {
		h.Reset()
		_, _ = h.Write(challenge)
		_, _ = h.Write(data)
		hash := h.Sum(hashBuf[:0])

		for i := 1; i < LabelHashNestingDepth; i++ {
			h.Reset()
			_, _ = h.Write(hash[:])
			hash = h.Sum(hash[:0])
		}
		return hash[:]
	}
}
