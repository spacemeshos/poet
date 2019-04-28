package main

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet/internal"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProverAndVerifier(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	x := make([]byte, 32)
	n := uint(33)

	_, err := rand.Read(x)
	assert.NoError(t, err)

	p, err := prover.New(x, n, shared.NewHashFunc(x))
	// defer p.DeleteStore()
	assert.NoError(t, err, "Failed to create prover")

	t.Log("Computing dag...")
	t1 := time.Now()

	phi, err := p.ComputeDag()
	assert.NoError(t, err, "Failed to compute dag")

	e := time.Since(t1)
	t.Logf("Proof generated in %s (%f) \n", e, e.Seconds())
	t.Logf("Dag root label: %s\n", internal.GetDisplayValue(phi))

	proof, err := p.GetNonInteractiveProof()
	assert.NoError(t, err, "Failed to create NIP")

	v, err := verifier.New(x, n, shared.NewHashFunc(x))
	assert.NoError(t, err, "Failed to create verifier")

	a, err := v.VerifyNIP(proof)
	assert.NoError(t, err, "Failed to verify NIP")
	assert.True(t, a, "Failed to verify NIP")

	c, err := v.CreateNipChallenge(proof.Phi)
	assert.NoError(t, err, "Failed to create NIP challenge")

	res := v.Verify(c, proof)
	assert.True(t, a, "Failed to verify NIP proof")

	c1, err := v.CreateRndChallenge()
	assert.NoError(t, err, "Failed to create rnd challenge")

	proof1, err := p.GetProof(c1)
	assert.NoError(t, err, "Failed to create interactive proof")

	res = v.Verify(c1, proof1)
	assert.True(t, res, "Failed to verify interactive proof")

	e1 := time.Since(t1)
	t.Logf("Proof verified in %s (%f)\n", e1-e, (e1 - e).Seconds())
}

func TestBigNip(t *testing.T) {
	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	const n = 10

	p, err := prover.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	phi, err := p.ComputeDag()
	fmt.Printf("Dag root label: %s\n", internal.GetDisplayValue(phi))
	assert.NoError(t, err)

	proof, err := p.GetNonInteractiveProof()
	assert.NoError(t, err)

	v, err := verifier.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreateNipChallenge(proof.Phi)
	assert.NoError(t, err)

	res := v.Verify(c, proof)
	assert.True(t, res, "failed to verify proof")

	p.DeleteStore()

}

func TestNip(t *testing.T) {
	var x = []byte("Spacemesh launched its mainent")
	const n = 11 // 33.6mb storage

	p, err := prover.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	phi, err := p.ComputeDag()
	fmt.Printf("Dag root label: %s\n", internal.GetDisplayValue(phi))
	assert.NoError(t, err)

	proof, err := p.GetNonInteractiveProof()
	assert.NoError(t, err)

	v, err := verifier.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreateNipChallenge(proof.Phi)
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

	p, err := prover.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	_, err = p.ComputeDag()

	//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
	assert.NoError(t, err)

	v, err := verifier.New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreateRndChallenge()
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

		p, err := prover.New(x, n, shared.NewScryptHashFunc(x))

		defer p.DeleteStore()

		assert.NoError(t, err)

		_, err = p.ComputeDag()

		//fmt.Printf("Dag root label: %s\n", GetDisplayValue(phi))
		assert.NoError(t, err)

		v, err := verifier.New(x, n, shared.NewScryptHashFunc(x))
		assert.NoError(t, err)

		for i := 0; i < 100; i++ {

			c, err := v.CreateRndChallenge()
			assert.NoError(t, err)

			proof, err := p.GetProof(c)
			assert.NoError(t, err)

			res := v.Verify(c, proof)

			if !res {
				println("Failed to verify proof. Challenge data:")
				c.Print()
				println("Proof:")
				internal.PrintProof(proof)
			}

			assert.True(t, res, "failed to verify proof")
		}
	}
}
