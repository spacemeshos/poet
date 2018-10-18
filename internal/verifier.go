package internal

import (
	"github.com/spacemeshos/poet-ref/shared"
)

type Challenge = shared.Challenge
type Proof = shared.Proof
type IVerifier = shared.IVerifier

type SMVerifier struct {
	x [] byte
	n int32
}

func (*SMVerifier) Verify(c Challenge, p Proof) bool {
	return true
}

func (*SMVerifier) CreteNipChallenge(phi []byte)  Challenge {
	return Challenge{}
}

// create a random challenge that can be used to challenge a prover
// that created a proof for shared params (x,t,n,w)
func (*SMVerifier) CreteRndChallenge() Challenge {
	// todo use crypto.random to generate T challenges (each n bits long)
	return Challenge{}
}


// Create a new verifier for commitment X and param n
func NewVerifier(x []byte, n int32) IVerifier {
	return &SMVerifier{x: x, n: n}
}



