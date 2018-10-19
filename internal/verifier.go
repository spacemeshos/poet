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

func (s *SMVerifier) CreteNipChallenge(phi []byte) (Challenge,error) {

	// γ := (Hx(φ,1),...Hx(φ,t))

	var data [shared.T]Identifier

	f := NewSMBinaryStringFactory()

	buf := new(bytes.Buffer)

	for i := 0; i < shared.T; i++ {

		buf.Reset()
		err := binary.Write(buf, binary.BigEndian, i)
		if err != nil {
			// return error here
			return Challenge{}, err
		}

		// pack (phi,t) into a bytes array
		d := append(phi, buf.Bytes()...)

		// Compute Hx(phi,t)
		b := s.h.Hash(d)

		// Convert hash to Uint64 and create a W bits long string
		bg := new(big.Int).SetBytes(b[:])
		v := bg.Uint64()
		s, _ := f.NewBinaryStringFromInt(v, shared.W)
		data[i] = Identifier(s.GetStringValue())
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
