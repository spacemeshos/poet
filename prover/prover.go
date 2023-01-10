package prover

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/sha256-simd"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/hash"
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

var (
	leavesPerSecondUpdateInterval = uint64(1_000_000)
	leavesPerSecond               = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "poet_leaves_per_second",
		Help: "Leaves calculation rate per second",
	})
	leavesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "poet_leaves",
		Help: "Number of generated leaves",
	})
)

func init() {
	lpsStr := os.Getenv("POET_LEAVES_PER_SECOND_INTERVAL")
	if lpsStr != "" {
		if lps, err := strconv.Atoi(lpsStr); err == nil {
			leavesPerSecondUpdateInterval = uint64(lps)
		} else {
			fmt.Printf("failed parsing POET_LEAVES_PER_SECOND_INTERVAL as int: %v", err)
		}
	}
}

var ErrShutdownRequested = errors.New("shutdown requested")

type persistFunc func(ctx context.Context, tree *merkle.Tree, treeCache *cache.Writer, nextLeafId uint64) error

var persist persistFunc = func(context.Context, *merkle.Tree, *cache.Writer, uint64) error { return nil }

// GenerateProof computes the PoET DAG, uses Fiat-Shamir to derive a challenge from the Merkle root and generates a Merkle
// proof using the challenge and the DAG.
func GenerateProof(
	ctx context.Context,
	datadir string,
	// labelHashFunc func(data []byte) []byte,
	statement []byte,
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
	defer treeCache.Close()

	return generateProof(ctx, statement, tree, treeCache, limit, 0, securityParam, persist)
}

// GenerateProofRecovery recovers proof generation, from a given 'nextLeafID' and for a given 'parkedNodes' snapshot.
func GenerateProofRecovery(
	ctx context.Context,
	datadir string,
	// labelHashFunc func(data []byte) []byte,
	statement []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	limit time.Time,
	securityParam uint8,
	nextLeafID uint64,
	parkedNodes [][]byte,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	treeCache, tree, err := makeRecoveryProofTree(ctx, datadir, merkleHashFunc, nextLeafID, parkedNodes)
	if err != nil {
		return 0, nil, err
	}
	defer treeCache.Close()

	return generateProof(ctx, statement, tree, treeCache, limit, nextLeafID, securityParam, persist)
}

// GenerateProofWithoutPersistency calls GenerateProof with disabled persistency functionality
// and potential soft/hard-shutdown recovery.
func GenerateProofWithoutPersistency(
	ctx context.Context,
	datadir string,
	// labelHashFunc func(data []byte) []byte,
	statement []byte,
	merkleHashFunc func(lChild, rChild []byte) []byte,
	limit time.Time,
	securityParam uint8,
	minMemoryLayer uint,
) (uint64, *shared.MerkleProof, error) {
	return GenerateProof(ctx, datadir, statement, merkleHashFunc, limit, securityParam, minMemoryLayer, persist)
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
		defer readWriter.Close()

		width, err := readWriter.Width()
		if err != nil {
			return nil, nil, err
		}

		// Each incremental layer divides the base layer by 2.
		expectedWidth := nextLeafID >> layer

		// If file is longer than expected, truncate the file.
		if expectedWidth < width {
			filename := filepath.Join(datadir, file)
			logging.FromContext(ctx).Sugar().Infof("Recovery: layer %v cache file width is ahead of the last known merkle tree state. expected: %d, found: %d. Truncating file...", layer, expectedWidth, width)
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
	if err != nil {
		return nil, nil, err
	}

	if err := tree.SetParkedNodes(parkedNodes); err != nil {
		return nil, nil, err
	}

	return treeCache, tree, nil
}

func MakeLabelFunc(challenge []byte) func(labelID uint64, leftSiblings [][]byte) []byte {
	hasher := sha256.New().(*sha256.Digest)
	var h [32]byte
	labelBuffer := make([]byte, 8)
	return func(labelID uint64, leftSiblings [][]byte) []byte {
		hasher.Reset()
		hasher.Write(challenge)

		binary.BigEndian.PutUint64(labelBuffer, labelID)
		hasher.Write(labelBuffer)

		for _, sibling := range leftSiblings {
			hasher.Write(sibling)
		}

		h = hasher.CheckSum()

		for i := 1; i < hash.LabelHashNestingDepth; i++ {
			hasher.Reset()
			hasher.Write(h[:])
			h = hasher.CheckSum()
		}
		return h[:]
	}
}

func doSequentialWork(
	ctx context.Context,
	// labelHashFunc func(data []byte) []byte,
	statement []byte,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	nextLeafID uint64,
	securityParam uint8,
	persist persistFunc,
) (uint64, error) {
	var parkedNodes [][]byte
	makeLabel := MakeLabelFunc(statement)
	leaves := nextLeafID
	lastLeaves := leaves
	lastLPSTimestamp := time.Now()
	logger := logging.FromContext(ctx)

	for leafID := nextLeafID; ; leafID++ {
		select {
		case <-ctx.Done():
			if err := persist(ctx, tree, treeCache, leafID); err != nil {
				return 0, fmt.Errorf("%w: error happened during persisting: %v", ErrShutdownRequested, err)
			}
			return leaves, nil
		default:
		}

		if leafID != 0 && leafID%hardShutdownCheckpointRate == 0 {
			if err := persist(ctx, tree, treeCache, leafID); err != nil {
				return 0, err
			}
		}

		if generated := leaves - lastLeaves; generated >= leavesPerSecondUpdateInterval {
			now := time.Now()
			lps := float64(leaves-lastLeaves) / now.Sub(lastLPSTimestamp).Seconds()
			lastLPSTimestamp = now
			logger.Info("calculated leaves per second", zap.Float64("rate", lps), zap.Uint64("total", leaves))
			leavesPerSecond.Set(lps)
			leavesCounter.Add(float64(leaves - lastLeaves))
			lastLeaves = leaves
		}

		// Generate the next leaf.
		parkedNodes = tree.GetParkedNodes(parkedNodes[:0])
		err := tree.AddLeaf(makeLabel(leafID, parkedNodes))
		if err != nil {
			return 0, err
		}
		leaves++
	}
}

func generateProof(
	ctx context.Context,
	// labelHashFunc func(data []byte) []byte,
	statement []byte,
	tree *merkle.Tree,
	treeCache *cache.Writer,
	end time.Time,
	nextLeafID uint64,
	securityParam uint8,
	persist persistFunc,
) (uint64, *shared.MerkleProof, error) {
	logger := logging.FromContext(ctx)
	logger.Info("generating proof", zap.Time("end", end), zap.Uint64("nextLeafID", nextLeafID))

	proofCtx, cancel := context.WithDeadline(ctx, end)
	defer cancel()
	leaves, err := doSequentialWork(proofCtx, statement, tree, treeCache, nextLeafID, securityParam, persist)
	if err != nil {
		return 0, nil, err
	}
	select {
	case <-ctx.Done():
		return 0, nil, ErrShutdownRequested
	default:
	}

	logging.FromContext(ctx).Sugar().Infof("Merkle tree construction finished with %d leaves, generating proof...", leaves)

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
