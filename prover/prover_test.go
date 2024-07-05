package prover

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

	challenge := []byte("challenge this")
	leavesCounter := prometheus.NewCounter(prometheus.CounterOpts{})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	persist := func(ctx context.Context, treeCache *cache.Writer, _ uint64) error {
		// Call GetReader() so that the cache would flush and validate structure.
		_, err := treeCache.GetReader()
		return err
	}

	treeCfg := TreeConfig{Datadir: t.TempDir(), MinMemoryLayer: 5}

	leaves, _, err := GenerateProof(
		ctx,
		leavesCounter,
		treeCfg,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(5*time.Second),
		150,
		persist,
	)
	r.ErrorIs(err, context.DeadlineExceeded)

	// recover without parked nodes
	leafs, merkleProof, err := GenerateProofRecovery(
		context.Background(),
		leavesCounter,
		treeCfg,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now(),
		150,
		leaves,
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

func TestTreeRecovery(t *testing.T) {
	t.Parallel()
	test := func(t *testing.T, leafs uint64) {
		t.Parallel()
		r := require.New(t)
		dataDir := t.TempDir()
		// Create a tree and add some nodes to it
		treeCfg := TreeConfig{
			MinMemoryLayer: 3,
			Datadir:        dataDir,
		}
		merklehashFuncFunc := hash.GenMerkleHashFunc([]byte{})
		tree, treeCache, err := makeProofTree(treeCfg, merklehashFuncFunc)
		r.NoError(err)

		for i := uint64(0); i < leafs; i++ {
			err := tree.AddLeaf(bytes.Repeat([]byte{byte(i)}, 32))
			r.NoError(err)
		}
		treeCache.Close()

		// recover the tree
		treeCache, recoveredTree, err := makeRecoveryProofTree(context.Background(), treeCfg, merklehashFuncFunc, leafs)
		r.NoError(err)
		defer treeCache.Close()
		// compare the trees by examining their roots and parked nodes
		r.Equal(tree.Root(), recoveredTree.Root())
		r.Equal(tree.GetParkedNodes(nil), recoveredTree.GetParkedNodes(nil))
	}

	t.Run("odd number of leafs", func(t *testing.T) {
		test(t, 1005)
	})
	t.Run("even number of leafs", func(t *testing.T) {
		test(t, 1000)
	})
}

func TestNewTreeFailOnDirectoryNotEmpty(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	// Create a tree and add some nodes to it
	treeCfg := TreeConfig{
		MinMemoryLayer: 3,
		Datadir:        t.TempDir(),
	}
	merklehashFuncFunc := hash.GenMerkleHashFunc([]byte{})
	tree, treeCache, err := makeProofTree(treeCfg, merklehashFuncFunc)
	r.NoError(err)
	r.NoError(tree.AddLeaf(bytes.Repeat([]byte{7}, 32)))
	treeCache.Close()

	// try to create a new tree in the same directory
	_, _, err = makeProofTree(treeCfg, merklehashFuncFunc)
	r.Error(err)
}
