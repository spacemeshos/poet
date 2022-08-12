package prover

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
)

const (
	// MerkleMinCacheLayer set the min layer in which all layers above will be cached, in addition to the base layer.
	MerkleMinCacheLayer = 0

	// LowestMerkleMinMemoryLayer set the lowest-allowed layer in which all layers above will be cached in-memory.
	LowestMerkleMinMemoryLayer = 1

	// The rate, in leaves, in which the proof generation state snapshot will be saved to disk
	// to allow potential crash recovery.
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

// GenerateProof computes the PoET DAG, uses Fiat-Shamir to derive a challenge from the Merkle root and generates a Merkle
// proof using the challenge and the DAG.
func GenerateProof(
	sig *signal.Signal,
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	limit time.Time,
	securityParam uint8,
	minMemoryLayer uint,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	tree, treeCache, err := makeProofTree(datadir, merkleHashFunc, minMemoryLayer)
	if err != nil {
		return 0, nil, err
	}
	return generateProof(sig, labelHashFunc, tree, treeCache, limit, 0, securityParam, persist)
}

// GenerateProofRecovery recovers proof generation, from a given 'nextLeafID' and for a given 'parkedNodes' snapshot.
func GenerateProofRecovery(
	sig *signal.Signal,
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	limit time.Time,
	securityParam uint8,
	nextLeafID uint64,
	parkedNodes [][]byte,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	treeCache, tree, err := makeRecoveryProofTree(datadir, merkleHashFunc, nextLeafID, parkedNodes)
	if err != nil {
		return 0, nil, err
	}
	return generateProof(sig, labelHashFunc, tree, treeCache, limit, nextLeafID, securityParam, persist)
}

// GenerateProofWithoutPersistency calls GenerateProof with disabled persistency functionality
// and potential soft/hard-shutdown recovery.
func GenerateProofWithoutPersistency(
	datadir string,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	limit time.Time,
	securityParam uint8,
	minMemoryLayer uint,
) (uint64, *shared.MerkleProof, error) {
	return GenerateProof(sig, datadir, labelHashFunc, merkleHashFunc, limit, securityParam, minMemoryLayer, persist)
}

func makeProofTree(
	datadir string,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	minMemoryLayer uint,
) (*merkle.Tree, *cache.Writer, error) {
	metaFactory := NewReadWriterMetaFactory(minMemoryLayer, datadir)

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
	nextLeafID uint64,
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
		expectedWidth := nextLeafID >> layer

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
	end time.Time,
	nextLeafID uint64,
	securityParam uint8,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	unblock := sig.BlockShutdown()
	defer unblock()

	makeLabel := shared.MakeLabelFunc()
	leaves := nextLeafID
	for leafID := nextLeafID; time.Until(end) > 0; leafID++ {
		// Handle persistence.
		if sig.ShutdownRequested {
			if err := persist(tree, treeCache, leafID); err != nil {
				return 0, nil, err
			}
			return 0, nil, ErrShutdownRequested
		} else if leafID != 0 && leafID%hardShutdownCheckpointRate == 0 {
			if err := persist(tree, treeCache, leafID); err != nil {
				return 0, nil, err
			}
		}

		// Generate the next leaf.
		err := tree.AddLeaf(makeLabel(labelHashFunc, leafID, tree.GetParkedNodes()))
		if err != nil {
			return 0, nil, err
		}
		leaves++
	}

	log.Info("Merkle tree construction finished with %d leaves, generating proof...", leaves)

	root := tree.Root()

	cacheReader, err := treeCache.GetReader()
	if err != nil {
		return 0, nil, err
	}
	log.Info("fiat shamir with root=%x, leaves=%d, security=%d", root, leaves, securityParam)
	provenLeafIndices := shared.FiatShamir(root, leaves, securityParam)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return 0, nil, err
	}

	return leaves, &shared.MerkleProof{
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
