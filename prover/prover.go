package prover

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/poet/shared"
)

const MerkleMinCacheLayer = 0 // Merkle nodes from this layer up will be cached in memory

func GetProof(challenge shared.Challenge, leafCount uint64, securityParam uint8) (shared.MerkleProof, error) {
	treeCache := cache.NewWriter(
		cache.Combine(
			cache.SpecificLayersPolicy(map[uint]bool{0: true}),
			cache.MinHeightPolicy(MerkleMinCacheLayer)),
		cache.MakeSliceReadWriterFactory())
	tree := merkle.NewTreeBuilder().WithHashFunc(challenge.MerkleHashFunc()).WithCacheWriter(treeCache).Build()

	for leafID := uint64(0); leafID < leafCount; leafID++ {
		err := tree.AddLeaf(shared.MakeLabel(challenge.LabelHashFunc(), leafID, tree.GetParkedNodes()))
		if err != nil {
			return shared.MerkleProof{}, err
		}
	}
	root := tree.Root()

	cacheReader, err := treeCache.GetReader()
	if err != nil {
		return shared.MerkleProof{}, err
	}
	provenLeafIndices := shared.FiatShamir(root, leafCount, securityParam)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return shared.MerkleProof{}, err
	}

	return shared.MerkleProof{
		Root:         root,
		ProvenLeaves: provenLeaves,
		ProofNodes:   proofNodes,
	}, nil
}
