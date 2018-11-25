package shared

import (
	"github.com/minio/sha256-simd" // simd optimized sha256 computation
	"hash"
)

// HashFunc implementation
type sha256Hash struct {
	x    []byte // arbitrary binary data
	hash hash.Hash
}

// Returns a new HashFunc Hx() for commitment X
func NewHashFunc(x []byte) HashFunc {
	return &sha256Hash{x: x, hash: sha256.New()}
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

func (h *sha256Hash) HashSingle(data []byte) []byte {
	h.hash.Reset()
	h.hash.Write(h.x)
	h.hash.Write(data)
	return h.hash.Sum([]byte{})
}
