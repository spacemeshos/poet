package shared

import (
	"context"
	"encoding/binary"
	"hash"

	"github.com/c0mm4nd/go-ripemd"
)

// FindSubmitPowNonce finds the nonce that solves the PoW challenge.
func FindSubmitPowNonce(
	ctx context.Context,
	powChallenge, poetChallenge, nodeID []byte,
	difficulty uint,
) (uint64, error) {
	var hash []byte
	p := NewPowHasher(powChallenge, nodeID, poetChallenge)

	for nonce := uint64(0); ; nonce++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		hash = p.Hash(nonce, hash[:0])

		if CheckLeadingZeroBits(hash, difficulty) {
			return nonce, nil
		}
	}
}

type powHasher struct {
	h     hash.Hash
	input []byte
}

func NewPowHasher(inputs ...[]byte) *powHasher {
	h := &powHasher{h: ripemd.New256(), input: []byte{}}
	for _, in := range inputs {
		h.input = append(h.input, in...)
	}
	h.input = append(h.input, make([]byte, 8)...) // placeholder for nonce
	return h
}

func (p *powHasher) Hash(nonce uint64, output []byte) []byte {
	nonceBytes := p.input[len(p.input)-8:]
	binary.LittleEndian.PutUint64(nonceBytes, nonce)

	p.h.Reset()
	p.h.Write(p.input)
	return p.h.Sum(output)
}

// CheckLeadingZeroBits checks if the first 'expected' bits of the byte array are all zero.
func CheckLeadingZeroBits(data []byte, expected uint) bool {
	if len(data)*8 < int(expected) {
		return false
	}
	for i := 0; i < int(expected/8); i++ {
		if data[i] != 0 {
			return false
		}
	}
	if expected%8 != 0 {
		if data[expected/8]>>(8-expected%8) != 0 {
			return false
		}
	}

	return true
}
