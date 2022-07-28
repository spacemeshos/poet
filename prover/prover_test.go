package prover

import (
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
	securityParam := shared.T
	duration := 10 * time.Millisecond
	leafs, _, err := GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), time.Now().Add(duration), securityParam, LowestMerkleMinMemoryLayer)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(leafs), "leafs/op")
}
