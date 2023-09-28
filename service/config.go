package service

const (
	defaultMemoryLayers             = 26 // Up to (1 << 26) * 2 - 1 Merkle tree cache nodes (32 bytes each) will be held in-memory
	defaultTreeFileBufferSize       = 4096
	defaultEstimatedLeavesPerSecond = 78000
)

type Config struct {
	// Merkle-Tree related configuration:
	EstimatedLeavesPerSecond uint `long:"lps"              description:"Estimated number of leaves generated per second"`
	MemoryLayers             uint `long:"memory"           description:"Number of top Merkle tree layers to cache in-memory"`
	TreeFileBufferSize       uint `long:"tree-file-buffer" description:"The size of memory buffer for file-based tree layers"`
}

func DefaultConfig() Config {
	return Config{
		EstimatedLeavesPerSecond: defaultEstimatedLeavesPerSecond,
		MemoryLayers:             defaultMemoryLayers,
		TreeFileBufferSize:       defaultTreeFileBufferSize,
	}
}
