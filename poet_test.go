package main

import (
	"crypto/rand"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProverAndVerifier(t *testing.T) {
	x := make([]byte, 32)
	n := uint(26)

	_, err := rand.Read(x)
	assert.NoError(t, err)

	p, err := internal.NewProver(x, n, shared.NewHashFunc(x))
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

	v, err := internal.NewVerifier(x, n, shared.NewHashFunc(x))
	assert.NoError(t, err, "Failed to create verifier")

	a, err := v.VerifyNIP(proof)
	assert.NoError(t, err, "Failed to verify NIP")
	assert.True(t, a, "Failed to verify NIP")

	c, err := v.CreteNipChallenge(proof.Phi)
	assert.NoError(t, err, "Failed to create NIP challenge")

	res := v.Verify(c, proof)
	assert.True(t, a, "Failed to verify NIP proof")

	c1, err := v.CreteRndChallenge()
	assert.NoError(t, err, "Failed to create rnd challenge")

	proof1, err := p.GetProof(c1)
	assert.NoError(t, err, "Failed to create interactive proof")

	res = v.Verify(c1, proof1)
	assert.True(t, res, "Failed to verify interactive proof")

	e1 := time.Since(t1)
	t.Logf("Proof verified in %s (%f)\n", e1-e, (e1 - e).Seconds())
}
