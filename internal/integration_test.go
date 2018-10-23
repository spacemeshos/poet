package internal

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBigNip(t *testing.T) {

	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	// with n=25 and 16GB rqm:
	// Map size:  67108863 entries ~20GB - runtime: 1034.77s
	const n = 11

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

func TestNip(t *testing.T) {

	var x = []byte("Spacemesh launched its mainent")
	const n = 11

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

	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	const n = 11

	p, err := NewProver(x, n)
	assert.NoError(t, err)

	p.ComputeDag(func(phi shared.Label, err error) {

		//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
		assert.NoError(t, err)

		v, err := NewVerifier(x, n)
		assert.NoError(t, err)

		c, err := v.CreteRndChallenge()
		assert.NoError(t, err)

		println("Challenge data:")
		c.Print()

		proof, err := p.GetProof(c)
		assert.NoError(t, err)
		// PrintProof(proof)

		res := v.Verify(c, proof)
		assert.True(t, res, "failed to verify proof")
	})
}

func TestRndChallengeProofEx(t *testing.T) {

	for i := 0; i < 10000; i++ {

		// generate random commitment
		x := make([]byte, 32)
		_, err := rand.Read(x)
		assert.NoError(t, err)

		const n = 9

		p, err := NewProver(x, n)
		assert.NoError(t, err)

		p.ComputeDag(func(phi shared.Label, err error) {

			//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
			assert.NoError(t, err)

			v, err := NewVerifier(x, n)
			assert.NoError(t, err)

			c, err := v.CreteRndChallenge()
			assert.NoError(t, err)

			//println("Challenge data:")
			//c.Print()

			proof, err := p.GetProof(c)
			assert.NoError(t, err)
			// PrintProof(proof)

			res := v.Verify(c, proof)
			assert.True(t, res, "failed to verify proof")
		})
	}
}
