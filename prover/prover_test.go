package prover

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
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

func TestRecoverParkedNodes(t *testing.T) {
	r := require.New(t)

	// override snapshot interval
	hardShutdownCheckpointRate = 1000

	challenge := []byte("challenge this")
	leavesCounter := prometheus.NewCounter(prometheus.CounterOpts{})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))
	defer cancel()

	limit := time.Now().Add(time.Second * 5)

	var lastParkedNodes [][]byte
	var lastLeafs uint64

	persist := func(ctx context.Context, tree *merkle.Tree, treeCache *cache.Writer, numLeaves uint64) error {
		// Call GetReader() so that the cache would flush and validate structure.
		if _, err := treeCache.GetReader(); err != nil {
			return err
		}
		lastParkedNodes = tree.GetParkedNodes(lastParkedNodes[:0])
		lastLeafs = numLeaves
		return nil
	}

	treeCfg := TreeConfig{Datadir: t.TempDir(), MinMemoryLayer: 5}

	_, _, err := GenerateProof(
		ctx,
		leavesCounter,
		treeCfg,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		limit,
		150,
		persist,
	)
	r.ErrorIs(err, context.DeadlineExceeded)

	for id, node := range lastParkedNodes {
		if len(node) > 0 {
			t.Logf("parked node %d: %X", id, node)
		}
	}

	// recover without parked nodes

	leafs, merkleProof, err := GenerateProofRecovery(
		context.Background(),
		leavesCounter,
		treeCfg,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		limit,
		150,
		lastLeafs,
		persist,
	)
	r.NoError(err)

	err = verifier.Validate(
		*merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		leafs,
		150,
	)
	r.NoError(err)
}
