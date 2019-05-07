package prover

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/spacemeshos/merkle-tree/cache/readwriters"
	"os"
)

type ReadWriterMetaFactory struct {
	minMemoryLayer uint
	filesCreated   map[string]bool
}

func NewReadWriterMetaFactory(minMemoryLayer uint) *ReadWriterMetaFactory {
	return &ReadWriterMetaFactory{
		minMemoryLayer: minMemoryLayer,
		filesCreated:   make(map[string]bool),
	}
}

func (mf *ReadWriterMetaFactory) GetFactory() cache.LayerFactory {
	return func(layerHeight uint) (cache.LayerReadWriter, error) {
		if layerHeight < mf.minMemoryLayer {
			fileName := makeFileName(layerHeight)
			readWriter, err := readwriters.NewFileReadWriter(fileName)
			if err != nil {
				return nil, err
			}
			mf.filesCreated[fileName] = true
			return readWriter, nil
		}
		return &readwriters.SliceReadWriter{}, nil
	}
}

func (mf *ReadWriterMetaFactory) Cleanup() {
	failedRemovals := make(map[string]bool)
	for filename := range mf.filesCreated {
		err := os.Remove(filename)
		if err != nil {
			log.Error("could not remove temp file %v: %v", filename, err)
			failedRemovals[filename] = true
		}
	}
	mf.filesCreated = failedRemovals
}

func makeFileName(layer uint) string {
	return fmt.Sprintf("poet_layercache_%d.bin", layer)
}
