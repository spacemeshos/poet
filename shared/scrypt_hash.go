package shared

import (
	"golang.org/x/crypto/scrypt"
)

const ( // scrypt params
	n = 1024
	r = 1
	p = 1
	k = 32 // key len
)

// HashFunc implementation
type scryptHash struct {
	x []byte // arbitrary binary data

}

// Returns a new HashFunc Hx() for commitment X
func NewScryptHashFunc(x []byte) HashFunc {
	return &scryptHash{
		x: x,
	}
}

// Hash implements Hx()
//func (h *sha256Hash) Hash(data []byte) [WB]byte {
//	return sha256.Sum256(append(h.x, data...))
//}

// Hash implements Hx()
func (h *scryptHash) Hash(data ...[]byte) []byte {

	var d []byte
	for _, i := range data {
		d = append(d, i...)
	}

	res, err := scrypt.Key(d, h.x, n, r, p, k)
	if err != nil { // this means bad params but we should really return an error here and not panic
		panic(err)
	}
	return res
}

func (h *scryptHash) HashSingle(data []byte) []byte {
	res, err := scrypt.Key(data, h.x, n, r, p, k)
	if err != nil {
		panic(err)
	}
	return res
}
