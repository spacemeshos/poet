package shared

import (
	"github.com/minio/sha256-simd" // simd optimized sha256 computation
	"hash"
)

// HashFunc implementation
type sha256Hash struct {
	x    []byte // arbitrary binary data
	hash hash.Hash
	iters int
	emptySlice []byte
}

// Returns a new HashFunc Hx() for commitment X
func NewHashFunc(x []byte) HashFunc {

	// todo: pick it form parsm
	iters := 50

	return &sha256Hash{x: x, hash: sha256.New(), iters: iters}
}

// Hash implements Hx()
//func (h *sha256Hash) Hash(data []byte) [WB]byte {
//	return sha256.Sum256(append(h.x, data...))
//}

// Hash implements Hx()
func (h *sha256Hash) Hash(data ...[]byte) []byte {
	h.hash.Reset()
	h.hash.Write(h.x)
	for _, d := range data {
		_, _ = h.hash.Write(d)
	}

	return h.hash.Sum([]byte{})
}

// Multiple iterations hash using client provided iters
func (h *sha256Hash) HashIters(data ...[]byte) []byte {

	h.hash.Reset()

	// first, hash x
	h.hash.Write(h.x)

	// hash all user provided data
	for _, d := range data {
		_, _ = h.hash.Write(d)
	}

	digest := h.hash.Sum([]byte{})

	// perform iter hashes of x and user data
	for i := 0; i < h.iters; i++ {
		h.hash.Reset()
		h.hash.Write(h.x)
		h.hash.Write(digest)
		digest = h.hash.Sum(h.emptySlice)
	}

	return digest
}


func (h *sha256Hash) HashSingle(data []byte) []byte {
	h.hash.Reset()
	h.hash.Write(h.x)
	h.hash.Write(data)
	return h.hash.Sum([]byte{})
}
