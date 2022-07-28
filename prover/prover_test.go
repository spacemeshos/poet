package prover

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/require"
)

func TestGetProof(t *testing.T) {
	r := require.New(t)
	tempdir, _ := ioutil.TempDir("", "poet-test")

	challenge := []byte("challenge this")
	leafs, merkleProof, err := GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), time.Now().Add(10*time.Millisecond), 5, LowestMerkleMinMemoryLayer)
	r.NoError(err)
	t.Logf("root: %x", merkleProof.Root)
	t.Logf("proof: %x", merkleProof.ProvenLeaves)
	t.Logf("leafs: %d", leafs)
}

func BenchmarkGetProof(b *testing.B) {
	tempdir := b.TempDir()

	challenge := []byte("challenge this! challenge this! ")
	numLeaves := uint64(1) << 20
	securityParam := shared.T
	fmt.Printf("=> Generating proof for %d leaves with security param %d...\n", numLeaves, securityParam)

	total := uint64(0)
	for i := 0; i < b.N; i++ {
		leafs, _, err := GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), time.Now().Add(100*time.Microsecond), securityParam, LowestMerkleMinMemoryLayer)
		if err != nil {
			b.Fatal(err)
		}
		total += leafs
	}
	b.ReportMetric(float64(total), "leafs")
}
