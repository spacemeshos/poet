package prover

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/poet/shared"
)

type MerkleProof struct {
	root         []byte
	provenLeaves [][]byte
	proofNodes   [][]byte
}

type NewChallenge interface {
	MerkleHashFunc() merkle.HashFunc
	LabelHashFunc() shared.LabelHashFunc
}

func GetProof(challenge NewChallenge, height uint) (MerkleProof, error) {
	treeCache := cache.NewWriter(
		cache.Combine(
			cache.SpecificLayersPolicy(map[uint]bool{0: true}),
			cache.MinHeightPolicy(11)),
		cache.MakeSliceReadWriterFactory())
	tree := merkle.NewTreeBuilder().WithHashFunc(challenge.MerkleHashFunc()).WithCacheWriter(treeCache).Build()

	leafCount := uint64(1) << height
	for leafID := uint64(0); leafID < leafCount; leafID++ {
		err := tree.AddLeaf(shared.MakeLabel(challenge.LabelHashFunc(), leafID, tree.GetParkedNodes()))
		if err != nil {
			return MerkleProof{}, err
		}
	}
	root := tree.Root()

	cacheReader, err := treeCache.GetReader()
	if err != nil {
		return MerkleProof{}, err
	}
	securityParam := uint8(5)
	provenLeafIndices := shared.FiatShamir(root, leafCount, securityParam)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, cacheReader)

	return MerkleProof{
		root:         root,
		provenLeaves: provenLeaves,
		proofNodes:   proofNodes,
	}, nil
}
