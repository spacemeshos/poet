package main

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

func BenchmarkProverAndVerifierBig(b *testing.B) {
	r := require.New(b)
	tempdir, _ := ioutil.TempDir("", "poet-test")

	challenge := make([]byte, 32)
	n := uint(33)

	_, err := rand.Read(challenge)
	r.NoError(err)

	numLeaves := uint64(1) << n
	securityParam := shared.T

	b.Log("Computing dag...")
	t1 := time.Now()
	merkleProof, err := prover.GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam, prover.LowestMerkleMinMemoryLayer)
	r.NoError(err, "Failed to generate proof")

	e := time.Since(t1)
	b.Logf("Proof generated in %s (%f) \n", e, e.Seconds())
	b.Logf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam)
	r.NoError(err, "Failed to verify NIP")

	e1 := time.Since(t1)
	b.Logf("Proof verified in %s (%f)\n", e1-e, (e1 - e).Seconds())
}

func TestNip(t *testing.T) {
	tempdir, _ := ioutil.TempDir("", "poet-test")
	challenge := []byte("Spacemesh launched its mainnet")
	const n = 15

	numLeaves := uint64(1) << n
	securityParam := shared.T

	merkleProof, err := prover.GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam, prover.LowestMerkleMinMemoryLayer)
	assert.NoError(t, err)
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam)
	assert.NoError(t, err, "failed to verify proof")
}

func BenchmarkProofEx(t *testing.B) {
	tempdir, _ := ioutil.TempDir("", "poet-test")
	for j := 0; j < 10; j++ {

		// generate random commitment
		challenge := make([]byte, 32)
		_, err := rand.Read(challenge)
		assert.NoError(t, err)

		const n = 15

		numLeaves := uint64(1) << n
		securityParam := shared.T

		merkleProof, err := prover.GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam, prover.LowestMerkleMinMemoryLayer)
		assert.NoError(t, err)
		fmt.Printf("Dag root label: %x\n", merkleProof.Root)

		err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam)
		assert.NoError(t, err, "failed to verify proof")
	}
}
