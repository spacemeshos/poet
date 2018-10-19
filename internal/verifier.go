package internal

import (
	"github.com/spacemeshos/poet-ref/shared"
	"math/big"
)

type Challenge = shared.Challenge
type Proof = shared.Proof
type IVerifier = shared.IVerifier
type Identifier = shared.Identifier
type HashFunc = shared.HashFunc

type SMVerifier struct {
	x []byte
	n uint
	h HashFunc
}

func (s *SMVerifier) Verify(c Challenge, p Proof) bool {
	return true
}

func (s *SMVerifier) CreteNipChallenge(phi []byte) Challenge {

	// γ := (Hx(φ,1),...Hx(φ,t))

	var data [shared.T]Identifier

	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		// set identifier
		b := s.h.Hash(phi) // todo: add i in bytes (bigEndian)

		bg := new(big.Int).SetBytes(b[:])
		v := bg.Uint64()

		s, _ := f.NewBinaryStringFromInt(v, shared.W)
		data[i] = Identifier(s.GetStringValue())
	}

	return Challenge{Data: data}
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
