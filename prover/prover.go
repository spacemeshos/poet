package prover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
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

type TreeConfig struct {
	MinMemoryLayer    uint
	Datadir           string
	FileWriterBufSize uint
}

type persistFunc func(ctx context.Context, tree *merkle.Tree, treeCache *cache.Writer, nextLeafId uint64) error

var persist persistFunc = func(context.Context, *merkle.Tree, *cache.Writer, uint64) error { return nil }

// GenerateProof computes the PoET DAG, uses Fiat-Shamir to derive a challenge from the Merkle root and generates a
// Merkle proof using the challenge and the DAG.
func GenerateProof(
	ctx context.Context,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	limit time.Time,
	securityParam uint8,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	tree, treeCache, err := makeProofTree(treeCfg, merkleHashFunc)
	if err != nil {
		return 0, nil, err
	}
	defer treeCache.Close()

	return generateProof(ctx, labelHashFunc, tree, treeCache, limit, 0, securityParam, persist)
}

// GenerateProofRecovery recovers proof generation, from a given 'nextLeafID' and for a given 'parkedNodes' snapshot.
func GenerateProofRecovery(
	ctx context.Context,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	limit time.Time,
	securityParam uint8,
	nextLeafID uint64,
	parkedNodes [][]byte,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	treeCache, tree, err := makeRecoveryProofTree(ctx, treeCfg, merkleHashFunc, nextLeafID, parkedNodes)
	if err != nil {
		return 0, nil, err
	}
	defer treeCache.Close()

	return generateProof(ctx, labelHashFunc, tree, treeCache, limit, nextLeafID, securityParam, persist)
}

// GenerateProofWithoutPersistency calls GenerateProof with disabled persistency functionality
// and potential soft/hard-shutdown recovery.
func GenerateProofWithoutPersistency(
	ctx context.Context,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	limit time.Time,
	securityParam uint8,
) (uint64, *shared.MerkleProof, error) {
	return GenerateProof(ctx, treeCfg, labelHashFunc, merkleHashFunc, limit, securityParam, persist)
}

func makeProofTree(treeCfg TreeConfig, merkleHashFunc merkle.HashFunc) (*merkle.Tree, *cache.Writer, error) {
	if treeCfg.MinMemoryLayer < LowestMerkleMinMemoryLayer {
		treeCfg.MinMemoryLayer = LowestMerkleMinMemoryLayer
	}
	metaFactory := NewReadWriterMetaFactory(treeCfg.MinMemoryLayer, treeCfg.Datadir, treeCfg.FileWriterBufSize)

	treeCache := cache.NewWriter(
		cache.Combine(
			cache.SpecificLayersPolicy(map[uint]bool{0: true}),
			cache.MinHeightPolicy(MerkleMinCacheLayer)),
		metaFactory.GetFactory(),
	)

	tree, err := merkle.NewTreeBuilder().WithHashFunc(merkleHashFunc).WithCacheWriter(treeCache).Build()
	if err != nil {
		return nil, nil, err
	}

	return tree, treeCache, nil
}

func makeRecoveryProofTree(
	ctx context.Context,
	treeCfg TreeConfig,
	merkleHashFunc merkle.HashFunc,
	nextLeafID uint64,
	parkedNodes [][]byte,
) (*cache.Writer, *merkle.Tree, error) {
	// Don't use memory cache. Just utilize the existing files cache.
	maxUint := ^uint(0)
	layerFactory := NewReadWriterMetaFactory(maxUint, treeCfg.Datadir, treeCfg.FileWriterBufSize).GetFactory()

	layersFiles, err := getLayersFiles(treeCfg.Datadir)
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
		defer readWriter.Close()

		width, err := readWriter.Width()
		if err != nil {
			return nil, nil, err
		}

		// Each incremental layer divides the base layer by 2.
		expectedWidth := nextLeafID >> layer

		// If file is longer than expected, truncate the file.
		if expectedWidth < width {
			filename := filepath.Join(treeCfg.Datadir, file)
			logging.FromContext(ctx).
				Warn("Recovery: cache file width is ahead of the last known merkle tree state. Truncating file",
					zap.String("file", filename),
					zap.Uint("layer", layer),
					zap.Uint64("expected", expectedWidth),
					zap.Uint64("actual", width))
			if err := os.Truncate(filename, int64(expectedWidth*merkle.NodeSize)); err != nil {
				return nil, nil, fmt.Errorf("failed to truncate file: %v", err)
			}
		}

		// If file is shorter than expected, proof cannot be recovered.
		if expectedWidth > width {
			return nil, nil, fmt.Errorf(
				"layer %d cache file invalid width. expected: %d, found: %d",
				layer,
				expectedWidth,
				width,
			)
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
	if err != nil {
		return nil, nil, err
	}

	if err := tree.SetParkedNodes(parkedNodes); err != nil {
		return nil, nil, err
	}

	return treeCache, tree, nil
}

func sequentialWork(
	ctx context.Context,
	labelHashFunc func(data []byte) []byte,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	end time.Time,
	nextLeafID uint64,
	securityParam uint8,
	persist persistFunc,
) (uint64, error) {
	var parkedNodes [][]byte
	makeLabel := shared.MakeLabelFunc()

	finished := time.NewTimer(time.Until(end))
	defer finished.Stop()

	leavesCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poet_leaves",
		Help: "Number of generated leaves",
	})
	leavesCounter.Add(float64(nextLeafID))

	if err := prometheus.Register(leavesCounter); err != nil {
		logging.FromContext(ctx).Error("failed to register prometheus leaves counter", zap.Error(err))
	}
	defer prometheus.Unregister(leavesCounter)

	for {
		// Generate the next leaf.
		parkedNodes = tree.GetParkedNodes(parkedNodes[:0])
		err := tree.AddLeaf(makeLabel(labelHashFunc, nextLeafID, parkedNodes))
		if err != nil {
			return 0, err
		}
		nextLeafID++
		leavesCounter.Inc()

		select {
		case <-ctx.Done():
			if err := persist(ctx, tree, treeCache, nextLeafID); err != nil {
				return 0, fmt.Errorf("persisting execution state: %w", err)
			}
			return nextLeafID, ctx.Err()
		case <-finished.C:
			if err := persist(ctx, tree, treeCache, nextLeafID); err != nil {
				return 0, fmt.Errorf("persisting execution state: %w", err)
			}
			return nextLeafID, nil
		default:
		}

		if nextLeafID%hardShutdownCheckpointRate == 0 {
			if err := persist(ctx, tree, treeCache, nextLeafID); err != nil {
				return 0, err
			}
		}
	}
}

func generateProof(
	ctx context.Context,
	labelHashFunc func(data []byte) []byte,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	end time.Time,
	nextLeafID uint64,
	securityParam uint8,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	logger := logging.FromContext(ctx)
	logger.Info("generating proof", zap.Time("end", end), zap.Uint64("nextLeafID", nextLeafID))

	leaves, err := sequentialWork(ctx, labelHashFunc, tree, treeCache, end, nextLeafID, securityParam, persist)
	if err != nil {
		return 0, nil, err
	}

	logger.Sugar().Infof("merkle tree construction finished with %d leaves, generating proof...", leaves)

	started := time.Now()
	root := tree.Root()
	cacheReader, err := treeCache.GetReader()
	if err != nil {
		return 0, nil, err
	}
	provenLeafIndices := shared.FiatShamir(root, leaves, securityParam)
	_, provenLeaves, proofNodes, err := merkle.GenerateProof(provenLeafIndices, cacheReader)
	if err != nil {
		return 0, nil, err
	}
	logger.Sugar().Infof("proof generated, it took: %v", time.Since(started))

	return leaves, &shared.MerkleProof{
		Root: root,
		ProvenLeaves: sliceMap(provenLeaves, func(leaf []byte) shared.Leaf {
			var pLeaf shared.Leaf
			copy(pLeaf[:], leaf)
			return pLeaf
		}),
		ProofNodes: sliceMap(proofNodes, func(node []byte) shared.Node {
			var pNode shared.Node
			copy(pNode[:], node)
			return pNode
		}),
	}, nil
}

func sliceMap[T, U any](s []T, f func(T) U) []U {
	r := make([]U, 0, len(s))
	for i := range s {
		r = append(r, f(s[i]))
	}
	return r
}

func getLayersFiles(datadir string) (map[uint]string, error) {
	entries, err := os.ReadDir(datadir)
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

// Calculate the root of a Merkle Tree with given leaves.
func CalcTreeRoot(leaves [][]byte) ([]byte, error) {
	tree, err := merkle.NewTree()
	if err != nil {
		return nil, fmt.Errorf("failed to generate tree: %w", err)
	}
	for _, member := range leaves {
		err := tree.AddLeaf(member)
		if err != nil {
			return nil, fmt.Errorf("failed to add leaf: %w", err)
		}
	}
	return tree.Root(), nil
}
