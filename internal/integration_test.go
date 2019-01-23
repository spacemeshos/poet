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

	const n = 10

	p, err := NewProver(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	phi, err := p.ComputeDag()
	fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
	assert.NoError(t, err)

	proof, err := p.GetNonInteractiveProof()
	assert.NoError(t, err)

	v, err := NewVerifier(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreteNipChallenge(proof.Phi)
	assert.NoError(t, err)

	res := v.Verify(c, proof)
	assert.True(t, res, "failed to verify proof")

	p.DeleteStore()

}

func TestNip(t *testing.T) {

	var x = []byte("Spacemesh launched its mainent")
	const n = 11 // 33.6mb storage

	p, err := NewProver(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	phi, err := p.ComputeDag()
	fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
	assert.NoError(t, err)

	proof, err := p.GetNonInteractiveProof()
	assert.NoError(t, err)

	v, err := NewVerifier(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreteNipChallenge(proof.Phi)
	assert.NoError(t, err)

	res := v.Verify(c, proof)
	assert.True(t, res, "failed to verify proof")

	p.DeleteStore()
}

func TestRndChallengeProof(t *testing.T) {

	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	const n = 9

	p, err := NewProver(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	_, err = p.ComputeDag()

	//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
	assert.NoError(t, err)

	v, err := NewVerifier(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreteRndChallenge()
	assert.NoError(t, err)

	// println("Challenge data:")
	// c.Print()

	proof, err := p.GetProof(c)
	assert.NoError(t, err)
	// PrintProof(proof)

	res := v.Verify(c, proof)
	assert.True(t, res, "failed to verify proof")

	p.DeleteStore()

}

func BenchmarkProofEx(t *testing.B) {

	for j := 0; j < 10; j++ {

		// generate random commitment
		x := make([]byte, 32)
		_, err := rand.Read(x)
		assert.NoError(t, err)

		const n = 15

		p, err := NewProver(x, n, shared.NewScryptHashFunc(x))

		defer p.DeleteStore()

		assert.NoError(t, err)

		_, err = p.ComputeDag()

		//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
		assert.NoError(t, err)

		v, err := NewVerifier(x, n, shared.NewScryptHashFunc(x))
		assert.NoError(t, err)

		for i := 0; i < 100; i++ {

			c, err := v.CreteRndChallenge()
			assert.NoError(t, err)

			proof, err := p.GetProof(c)
			assert.NoError(t, err)

			res := v.Verify(c, proof)

			if !res {
				println("Failed to verify proof. Challenge data:")
				c.Print()
				println("Proof:")
				PrintProof(proof)
			}

			assert.True(t, res, "failed to verify proof")
		}

	}
}
