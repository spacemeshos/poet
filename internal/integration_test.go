package internal

import (
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
const x = "this is a commitment"
const n = 25
Map size:  67108863 ~20GB
Computed root label: 68b4c66918faa1a6538920944f13957354910f741a87236ea4905f2a50314c10
PASS: TestProverBasic (1034.77s)
*/

func TestNip(t *testing.T) {

	var x = []byte("this is a commitment")
	const n = 2

	p, err := NewProver(x, n)
	assert.NoError(t, err)

	p.ComputeDag(func(phi shared.Label, err error) {
		fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
		assert.NoError(t, err)

		proof, err := p.GetNonInteractiveProof()
		assert.NoError(t, err)

		v, err := NewVerifier(x, n)
		assert.NoError(t, err)

		c, err := v.CreteNipChallenge(proof.Phi)
		assert.NoError(t, err)

		res := v.Verify(c, proof)
		assert.True(t, res, "failed to verify proof")
	})
}

func TestRndChallengeProof(t *testing.T) {

	var x = []byte("this is a commitment")
	const n = 2

	p, err := NewProver(x, n)
	assert.NoError(t, err)

	p.ComputeDag(func(phi shared.Label, err error) {
		fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
		assert.NoError(t, err)

		v, err := NewVerifier(x, n)
		assert.NoError(t, err)

		c, err := v.CreteRndChallenge()
		assert.NoError(t, err)

		proof, err := p.GetProof(c)
		assert.NoError(t, err)

		res := v.Verify(c, proof)
		assert.True(t, res, "failed to verify proof")
	})
}
