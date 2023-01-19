package prover

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

func TestGetProof(t *testing.T) {
	r := require.New(t)

	challenge := []byte("challenge this")
	leafs, merkleProof, err := GenerateProofWithoutPersistency(
		context.Background(),
		TreeConfig{Datadir: t.TempDir()},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(100*time.Millisecond),
		5,
	)
	r.NoError(err)
	t.Logf("root: %x", merkleProof.Root)
	t.Logf("proof: %x", merkleProof.ProvenLeaves)
	t.Logf("leafs: %d", leafs)

	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafs, 5)
	r.NoError(err)
}

func BenchmarkGetProof(b *testing.B) {
	challenge := []byte("challenge this! challenge this! ")
	securityParam := shared.T
	duration := time.Second * 30
	leafs, _, err := GenerateProofWithoutPersistency(
		context.Background(),
		TreeConfig{Datadir: b.TempDir()},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(duration),
		securityParam,
	)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(leafs)/duration.Seconds(), "leafs/sec")
	b.ReportMetric(float64(leafs), "leafs/op")
	b.SetBytes(int64(leafs) * 32)
}
