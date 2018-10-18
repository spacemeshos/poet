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

func NewVerifier(x []byte, n int32) IVerifier {
	v := &SMVerifier{
		x: x, n: n,
	}

	return v
}
