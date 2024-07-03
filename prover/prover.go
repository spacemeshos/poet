package prover

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	mshared "github.com/spacemeshos/merkle-tree/shared"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/maps"

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

type persistFunc func(ctx context.Context, treeCache *cache.Writer, nextLeafId uint64) error

// GenerateProof generates the Proof of Sequential Work. It stops when the given deadline is reached.
func GenerateProof(
	ctx context.Context,
	leavesCounter prometheus.Counter,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	deadline time.Time,
	securityParam uint8,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	tree, treeCache, err := makeProofTree(treeCfg, merkleHashFunc)
	if err != nil {
		return 0, nil, err
	}
	defer treeCache.Close()

	return generateProof(
		ctx,
		leavesCounter,
		labelHashFunc,
		tree,
		treeCache,
		deadline,
		0,
		securityParam,
		persist,
	)
}

// GenerateProofRecovery recovers proof generation, from a given 'nextLeafID'.
func GenerateProofRecovery(
	ctx context.Context,
	leavesCounter prometheus.Counter,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	deadline time.Time,
	securityParam uint8,
	nextLeafID uint64,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	treeCache, tree, err := makeRecoveryProofTree(ctx, treeCfg, merkleHashFunc, nextLeafID)
	if err != nil {
		return 0, nil, err
	}
	defer treeCache.Close()

	return generateProof(
		ctx,
		leavesCounter,
		labelHashFunc,
		tree,
		treeCache,
		deadline,
		nextLeafID,
		securityParam,
		persist,
	)
}

// GenerateProofWithoutPersistency calls GenerateProof with disabled persistency functionality.
// Tree recovery will not be possible. Meant to be used for testing purposes only.
// It doesn't expose metrics too.
func GenerateProofWithoutPersistency(
	ctx context.Context,
	treeCfg TreeConfig,
	labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc,
	deadline time.Time,
	securityParam uint8,
) (uint64, *shared.MerkleProof, error) {
	leavesCounter := prometheus.NewCounter(prometheus.CounterOpts{})
	return GenerateProof(
		ctx,
		leavesCounter,
		treeCfg,
		labelHashFunc,
		merkleHashFunc,
		deadline,
		securityParam,
		func(context.Context, *cache.Writer, uint64) error { return nil },
	)
}

// Create a new merkle tree for proof generation.
func makeProofTree(treeCfg TreeConfig, merkleHashFunc merkle.HashFunc) (*merkle.Tree, *cache.Writer, error) {
	// Make sure there are no existing layer files.
	layersFiles, err := getLayersFiles(treeCfg.Datadir)
	if err != nil {
		return nil, nil, err
	}
	if len(layersFiles) > 0 {
		return nil, nil, fmt.Errorf("datadir is not empty. %v layer files exist", maps.Values(layersFiles))
	}

	minMemoryLayer := max(treeCfg.MinMemoryLayer, LowestMerkleMinMemoryLayer)
	treeCache := cache.NewWriter(
		cache.Combine(
			cache.SpecificLayersPolicy(map[uint]bool{0: true}),
			cache.MinHeightPolicy(MerkleMinCacheLayer)),
		GetLayerFactory(minMemoryLayer, treeCfg.Datadir, treeCfg.FileWriterBufSize),
	)

	tree, err := merkle.NewTreeBuilder().WithHashFunc(merkleHashFunc).WithCacheWriter(treeCache).Build()
	if err != nil {
		return nil, nil, err
	}

	return tree, treeCache, nil
}

// Recover merkle tree from existing cache files.
func makeRecoveryProofTree(
	ctx context.Context,
	treeCfg TreeConfig,
	merkleHashFunc merkle.HashFunc,
	nextLeafID uint64,
) (*cache.Writer, *merkle.Tree, error) {
	// Don't use memory cache. Just utilize the existing files cache.
	layerFactory := GetLayerFactory(math.MaxUint, treeCfg.Datadir, treeCfg.FileWriterBufSize)

	layersFiles, err := getLayersFiles(treeCfg.Datadir)
	if err != nil {
		return nil, nil, err
	}

	// Validate that layer 0 exists.
	if _, ok := layersFiles[0]; !ok {
		return nil, nil, errors.New("layer 0 cache file is missing")
	}

	var topLayer uint
	parkedNodesMap := make(map[uint][]byte)

	// Validate structure.
	for layer, file := range layersFiles {
		topLayer = max(topLayer, layer)

		readWriter, err := layerFactory(layer)
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

		// recover parked node
		if expectedWidth%2 != 0 {
			if err := readWriter.Seek(expectedWidth - 1); err != nil {
				return nil, nil, fmt.Errorf("seeking to parked node in layer %d: %w", layer, err)
			}
			parkedNode, err := readWriter.ReadNext()
			if err != nil {
				return nil, nil, fmt.Errorf("reading parked node in layer %d: %w", layer, err)
			}
			parkedNodesMap[layer] = parkedNode
		}
	}

	// turn parkedNodesMap into a slice ordered by key
	var parkedNodes [][]byte
	for layer := 0; layer < len(layersFiles); layer++ {
		if node, ok := parkedNodesMap[uint(layer)]; ok {
			parkedNodes = append(parkedNodes, node)
		} else {
			parkedNodes = append(parkedNodes, []byte{})
		}
	}
	layerReader, err := layerFactory(topLayer)
	if err != nil {
		return nil, nil, err
	}
	defer layerReader.Close()
	memCachedParkedNodes, readCache, err := recoverMemCachedParkedNodes(layerReader, merkleHashFunc)
	if err != nil {
		return nil, nil, fmt.Errorf("recovering parked nodes from top layer of disk-cache: %w", err)
	}
	parkedNodes = append(parkedNodes, memCachedParkedNodes...)

	logging.FromContext(ctx).Debug("recovered parked nodes",
		zap.Array("nodes", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			for _, node := range parkedNodes {
				enc.AppendString(fmt.Sprintf("%X", node))
			}
			return nil
		})))

	layers := make(map[uint]bool)
	for layer := range layersFiles {
		layers[layer] = true
	}

	treeCache := cache.NewWriter(
		cache.SpecificLayersPolicy(layers),
		layerFactory)

	// populate layers from topLayer up with layers from the rebuilt tree
	rebuildLayers := readCache.Layers()
	for layer, reader := range rebuildLayers {
		if layer == 0 {
			continue
		}
		treeCache.SetLayer(topLayer+uint(layer), reader)
	}

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
	leavesCounter prometheus.Counter,
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

	stop := make(chan struct{})

	leavesCounter.Add(float64(nextLeafID))

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
			if err := persist(ctx, treeCache, nextLeafID); err != nil {
				return 0, fmt.Errorf("persisting execution state: %w", err)
			}
			return nextLeafID, ctx.Err()
		case <-finished.C:
			close(stop)
			if nextLeafID < uint64(securityParam) {
				// we reached deadline, but we didn't generate enough leaves to generate a valid proof, so continue
				// generating leaves until we have enough.
				continue
			}
			if err := persist(ctx, treeCache, nextLeafID); err != nil {
				return 0, fmt.Errorf("persisting execution state: %w", err)
			}
			return nextLeafID, nil
		case <-stop:
			if nextLeafID < uint64(securityParam) {
				// we reached deadline, but we didn't generate enough leaves to generate a valid proof, so continue
				// generating leaves until we have enough.
				continue
			}
			if err := persist(ctx, treeCache, nextLeafID); err != nil {
				return 0, fmt.Errorf("persisting execution state: %w", err)
			}
			return nextLeafID, nil
		default:
		}

		if nextLeafID%hardShutdownCheckpointRate == 0 {
			if err := persist(ctx, treeCache, nextLeafID); err != nil {
				return 0, err
			}
		}
	}
}

func generateProof(
	ctx context.Context,
	leavesCounter prometheus.Counter,
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

	leaves, err := sequentialWork(
		ctx,
		leavesCounter,
		labelHashFunc,
		tree,
		treeCache,
		end,
		nextLeafID,
		securityParam,
		persist,
	)
	if err != nil {
		return leaves, nil, err
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
		Root:         root,
		ProvenLeaves: provenLeaves,
		ProofNodes:   proofNodes,
	}, nil
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

// build a small tree with the nodes from the top layer of the cache as leafs.
// this tree will be used to get parked nodes for the merkle tree.
func recoverMemCachedParkedNodes(
	layerReader mshared.LayerReader,
	merkleHashFunc merkle.HashFunc,
) ([][]byte, mshared.CacheReader, error) {
	recoveryTreeCache := cache.NewWriter(func(uint) bool { return true }, GetLayerFactory(0, "", 0))

	tree, err := merkle.NewTreeBuilder().WithHashFunc(merkleHashFunc).WithCacheWriter(recoveryTreeCache).Build()
	if err != nil {
		return nil, nil, err
	}

	// append nodes as leafs
	for {
		node, err := layerReader.ReadNext()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading node from top layer of disk-cache: %w", err)
		}
		if err := tree.AddLeaf(node); err != nil {
			return nil, nil, fmt.Errorf("adding node to small tree: %w", err)
		}
	}
	rdr, err := recoveryTreeCache.GetReader()
	if err != nil {
		return nil, nil, fmt.Errorf("getting reader for small tree: %w", err)
	}
	// the first parked node is for the leaves from the layerReader.
	return tree.GetParkedNodes(nil)[1:], rdr, nil
}
