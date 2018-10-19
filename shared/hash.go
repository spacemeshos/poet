package shared

import (
	"github.com/minio/sha256-simd" // simd optimized sha256 computation
)

// HashFunc implementation
type sha256Hash struct {
	x []byte // arbitrary binary data
}

// Returns a new HashFunc Hx()
func NewHashFunc(x []byte) HashFunc {
	h := &sha256Hash{x: x}
	return h
}

// Hash implements Hx()
func (h *sha256Hash) Hash(data []byte) [WB]byte {

	//
	// todo: benchmark the append here - it is performed on every hash
	// ... It is likely that append() copies 1 byte at a time
	//
	return sha256.Sum256(append(h.x, data...))
}
