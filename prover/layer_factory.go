package prover

import (
	"fmt"
	"path/filepath"

	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/merkle-tree/cache/readwriters"
)

// GetLayerFactory creates a merkle LayerFactory.
// The minMemoryLayer determines the threshold below which layers are saved on-disk, while layers equal and above -
// in-memory.
func GetLayerFactory(minMemoryLayer uint, datadir string, fileWriterBufSize uint) cache.LayerFactory {
	return func(layerHeight uint) (cache.LayerReadWriter, error) {
		if layerHeight < minMemoryLayer {
			fileName := filepath.Join(datadir, fmt.Sprintf("layercache_%d.bin", layerHeight))
			readWriter, err := readwriters.NewFileReadWriter(fileName, int(fileWriterBufSize))
			if err != nil {
				return nil, err
			}

			return readWriter, nil
		}
		return &readwriters.SliceReadWriter{}, nil
	}
}
