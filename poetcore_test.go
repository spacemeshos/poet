package main

import (
	"crypto/rand"
	"fmt"
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

	challenge := shared.Sha256Challenge(x)
	leafCount := uint64(1) << n
	securityParam := shared.T

	t.Log("Computing dag...")
	t1 := time.Now()
	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	assert.NoError(t, err, "Failed to generate proof")

	e := time.Since(t1)
	t.Logf("Proof generated in %s (%f) \n", e, e.Seconds())
	t.Logf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(merkleProof, challenge, leafCount, securityParam)
	assert.NoError(t, err, "Failed to verify NIP")

	e1 := time.Since(t1)
	t.Logf("Proof verified in %s (%f)\n", e1-e, (e1 - e).Seconds())
}

func TestBigNip(t *testing.T) {
	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	const n = 10

	challenge := shared.Sha256Challenge(x)
	leafCount := uint64(1) << n
	securityParam := shared.T

	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	assert.NoError(t, err)
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(merkleProof, challenge, leafCount, securityParam)
	assert.NoError(t, err, "failed to verify proof")
}

func TestNip(t *testing.T) {
	var x = []byte("Spacemesh launched its mainnet")
	const n = 11 // 33.6mb storage

	challenge := shared.Sha256Challenge(x)
	leafCount := uint64(1) << n
	securityParam := shared.T

	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	assert.NoError(t, err)
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(merkleProof, challenge, leafCount, securityParam)
	assert.NoError(t, err, "failed to verify proof")
}

func TestRndChallengeProof(t *testing.T) {
	x := make([]byte, 32)
	_, err := rand.Read(x)
	assert.NoError(t, err)

	const n = 9

	challenge := shared.Sha256Challenge(x)
	leafCount := uint64(1) << n
	securityParam := shared.T

	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	assert.NoError(t, err)
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(merkleProof, challenge, leafCount, securityParam)
	assert.NoError(t, err, "failed to verify proof")
}

func BenchmarkProofEx(t *testing.B) {
	for j := 0; j < 10; j++ {

		// generate random commitment
		x := make([]byte, 32)
		_, err := rand.Read(x)
		assert.NoError(t, err)

		const n = 15

		challenge := shared.Sha256Challenge(x)
		leafCount := uint64(1) << n
		securityParam := shared.T

		merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
		assert.NoError(t, err)
		fmt.Printf("Dag root label: %x\n", merkleProof.Root)

		err = verifier.Validate(merkleProof, challenge, leafCount, securityParam)
		assert.NoError(t, err, "failed to verify proof")
	}
}
