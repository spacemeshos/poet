package prover

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	MerkleMinCacheLayer        = 0 // Merkle nodes from this layer up will be cached, in addition to the base layer
	MerkleMinMemoryLayer       = 2 // Below this layer caching is done on-disk, from this layer up -- in-memory
	hardShutdownCheckpointRate = 1 << 24
)

var (
	ErrShutdownRequested = errors.New("shutdown requested")
)

type persistFunc func(tree *merkle.Tree, treeCache *cache.Writer, nextLeafId uint64) error

var (
	sig                 = signal.NewSignal()
	persist persistFunc = func(tree *merkle.Tree, treeCache *cache.Writer, nextLeafId uint64) error { return nil }
)

func GenerateProofWithoutPersistency(
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	numLeaves uint64,
	securityParam uint8,
) (*shared.MerkleProof, error) {
	return GenerateProof(sig, datadir, labelHashFunc, merkleHashFunc, numLeaves, securityParam, persist)
}

// GenerateProof computes the PoET DAG, uses Fiat-Shamir to derive a challenge from the Merkle root and generates a Merkle
// proof using the challenge and the DAG.
func GenerateProof(
	sig *signal.Signal,
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	numLeaves uint64,
	securityParam uint8,
	persist persistFunc,
) (*shared.MerkleProof, error) {
	tree, treeCache, err := makeProofTree(datadir, merkleHashFunc)
	if err != nil {
		return nil, err
	}

	return generateProof(sig, labelHashFunc, tree, treeCache, numLeaves, 0, securityParam, persist)
}

func GenerateProofRecovery(
	sig *signal.Signal,
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	numLeaves uint64,
	securityParam uint8,
	nextLeafId uint64,
	parkedNodes [][]byte,
	persist persistFunc,
) (*shared.MerkleProof, error) {
	treeCache, tree, err := makeRecoveryProofTree(datadir, merkleHashFunc, nextLeafId, parkedNodes)
	if err != nil {
		return nil, err
	}

	return generateProof(sig, labelHashFunc, tree, treeCache, numLeaves, nextLeafId, securityParam, persist)
}

func makeProofTree(datadir string, merkleHashFunc func(lChild, rChild []byte) []byte) (*merkle.Tree, *cache.Writer, error) {
	metaFactory := NewReadWriterMetaFactory(MerkleMinMemoryLayer, datadir)

	treeCache := cache.NewWriter(
		cache.Combine(
			cache.SpecificLayersPolicy(map[uint]bool{0: true}),
			cache.MinHeightPolicy(MerkleMinCacheLayer)),
		metaFactory.GetFactory())

	tree, err := merkle.NewTreeBuilder().WithHashFunc(merkleHashFunc).WithCacheWriter(treeCache).Build()
	if err != nil {
		return nil, nil, err
	}

	return tree, treeCache, nil
}

func makeRecoveryProofTree(
	datadir string,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	nextLeafId uint64,
	parkedNodes [][]byte,
) (*cache.Writer, *merkle.Tree, error) {

	// Don't use memory cache. Just utilize the existing files cache.
	maxUint := ^uint(0)
	layerFactory := NewReadWriterMetaFactory(maxUint, datadir).GetFactory()

	layersFiles, err := getLayersFiles(datadir)
	if err != nil {
		return nil, nil, err
	}

	// Validate that layer 0 exists.
	_, ok := layersFiles[0]
	if !ok {
		return nil, nil, fmt.Errorf("layer 0 cache file is missing")
	}

	// Validate structure.
	for layer, file := range layersFiles {
		readWriter, err := layerFactory(uint(layer))
		if err != nil {
			return nil, nil, err
		}
		width, err := readWriter.Width()
		if err != nil {
			return nil, nil, err
		}

		// Each incremental layer divides the base layer by 2.
		expectedWidth := nextLeafId >> layer

		// If file is longer than expected, truncate the file.
		if expectedWidth < width {
			filename := filepath.Join(datadir, file)
			log.Info("Recovery: layer %v cache file width is ahead of the last known merkle tree state. expected: %d, found: %d. Truncating file...", layer, expectedWidth, width)
			if err := os.Truncate(filename, int64(expectedWidth*merkle.NodeSize)); err != nil {
				return nil, nil, fmt.Errorf("failed to truncate file: %v", err)
			}
		}

		// If file is shorter than expected, proof cannot be recovered.
		if expectedWidth > width {
			return nil, nil, fmt.Errorf("layer %d cache file invalid width. expected: %d, found: %d", layer, expectedWidth, width)
		}
	}

	layers := make(map[uint]bool)
	for layer := range layersFiles {
		layers[layer] = true
	}

	treeCache := cache.NewWriter(
		cache.SpecificLayersPolicy(layers),
		layerFactory)

	tree, err := merkle.NewTreeBuilder().
		WithHashFunc(merkleHashFunc).
		WithCacheWriter(treeCache).
		Build()

	if err := tree.SetParkedNodes(parkedNodes); err != nil {
		return nil, nil, err
	}

	return treeCache, tree, nil
}

func generateProof(
	sig *signal.Signal,
	labelHashFunc func(data []byte) []byte,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	numLeaves uint64,
	nextLeafId uint64,
	securityParam uint8,
	persist persistFunc,
) (*shared.MerkleProof, error) {
	unblock := sig.BlockShutdown()
	defer unblock()

	for leafId := nextLeafId; leafId < numLeaves; leafId++ {
		// Handle persistence.
		if sig.ShutdownRequested {
			if err := persist(tree, treeCache, leafId); err != nil {
				return nil, err
			}
			return nil, ErrShutdownRequested
		} else if leafId != 0 && leafId%hardShutdownCheckpointRate == 0 {
			if err := persist(tree, treeCache, leafId); err != nil {
				return nil, err
			}
		}

		// Generate the next leaf.
		err := tree.AddLeaf(shared.MakeLabel(labelHashFunc, leafId, tree.GetParkedNodes()))
		if err != nil {
			return nil, err
		}
	}
	root := tree.Root()

	cacheReader, err := treeCache.GetReader()
	if err != nil {
		return nil, err
	}
	provenLeafIndices := shared.FiatShamir(root, numLeaves, securityParam)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return nil, err
	}

	return &shared.MerkleProof{
		Root:         root,
		ProvenLeaves: provenLeaves,
		ProofNodes:   proofNodes,
	}, nil

}

func getLayersFiles(datadir string) (map[uint]string, error) {
	entries, err := ioutil.ReadDir(datadir)
	if err != nil {
		return nil, err
	}

	files := make(map[uint]string, 0)
	for _, entry := range entries {
		prefix := "layercache_"
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, prefix) {
			re := regexp.MustCompile("layercache_(.*).bin")
			matches := re.FindStringSubmatch(name)
			if len(matches) != 2 {
				return nil, fmt.Errorf("invalid layer cache filename: %v", name)
			}
			layerNum, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("invalid layer cache filename: %v", name)
			}

			files[uint(layerNum)] = name
		}
	}

	return files, nil
}
