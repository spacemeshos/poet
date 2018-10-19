package internal

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
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

	for i := 0; i < shared.T; i++ {
		// set identifier
		data[i] = "010101"
	}

	return Challenge{Data: data}
}

// create a random challenge that can be used to challenge a prover
// that created a proof for shared params (x,t,n,w)
func (s *SMVerifier) CreteRndChallenge() (Challenge, error) {

	// todo use crypto.random to generate T challenges (each n bits long)
	var data [shared.T]Identifier

	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {

		// generate l random bytes
		var l = s.n / 8
		if s.n%8 != 0 {
			l++
		}

		buff := make([]byte, l)
		_, err := rand.Read(buff)

		if err != nil {
			fmt.Println("error:", err)
			// throw error here
			return Challenge{}, err
		}

		// create a binary string
		b, err := f.NewBinaryStringFromInt(0, s.n)
		if err != nil {
			return Challenge{}, err
		}

		data[i] = Identifier(b.GetStringValue())
	}

	c := Challenge{data}

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
