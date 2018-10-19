package internal

import (
	"bytes"
	"encoding/binary"
	"github.com/spacemeshos/poet-ref/shared"
	"math/big"
)

type Challenge = shared.Challenge
type Proof = shared.Proof
type IVerifier = shared.IVerifier
type Identifier = shared.Identifier
type HashFunc = shared.HashFunc

type SMVerifier struct {
	x []byte	// commitment
	n uint		// n param 1 <= n <= 63
	h HashFunc	// Hx()
}

func (s *SMVerifier) Verify(c Challenge, p Proof) bool {
	return true
}

func (s *SMVerifier) CreteNipChallenge(phi []byte) (Challenge, error) {

	// γ := (Hx(φ,1),...Hx(φ,t))

	var data [shared.T]Identifier

	buf := new(bytes.Buffer)

	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		buf.Reset()
		err := binary.Write(buf, binary.BigEndian, uint8(i))
		if err != nil {
			// return error here
			return Challenge{}, err
		}

		// pack (phi, i) into a bytes array
		// Hx(phi, i) := Hx(phi ... bigEndianEncodedBytes(i))
		d := append(phi, buf.Bytes()...)

		// Compute Hx(phi, i)
		hash := s.h.Hash(d)

		// we take the first 64 bits from the hash
		bg := new(big.Int).SetBytes(hash[0:8])

		// Integer representation of the first 8 bytes
		v := bg.Uint64()

		// encode v as a 64 bits binary string
		bs, err := f.NewBinaryStringFromInt(v, 64)
		str := bs.GetStringValue()

		// take first s.n bits from the 64 bits binary string
		l := uint(len(str))
		if l > s.n {
			str = str[0 : s.n]
		}

		data[i] = Identifier(str)
	}

	return Challenge{Data: data}, nil
}

// create a random challenge that can be used to challenge a prover
// that created a proof for shared params (x,t,n,w)
func (s *SMVerifier) CreteRndChallenge() (Challenge, error) {

	var data [shared.T]Identifier

	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		b, err := f.NewRandomBinaryString(s.n)

		if err != nil {
			return Challenge{}, err
		}

		data[i] = Identifier(b.GetStringValue())
	}

	c := Challenge{Data: data}

	return c, nil
}

// Create a new verifier for commitment X and param n
func NewVerifier(x []byte, n uint) IVerifier {
	return &SMVerifier{
		x: x,
		n: n,
		h: shared.NewHashFunc(x),
	}
}
