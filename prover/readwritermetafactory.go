package prover

import (
	"fmt"
	"github.com/spacemeshos/merkle-tree/cache"
	"os"
)

type ReadWriterMetaFactory struct {
	minMemoryLayer uint
	filesCreated   []string
}

func NewReadWriterMetaFactory(minMemoryLayer uint) *ReadWriterMetaFactory {
	return &ReadWriterMetaFactory{minMemoryLayer: minMemoryLayer}
}

func (mf *ReadWriterMetaFactory) GetFactory() cache.LayerFactory {
	return func(layerHeight uint) (cache.LayerReadWriter, error) {
		if layerHeight < mf.minMemoryLayer {
			fileName := makeFileName(layerHeight)
			readWriter, err := NewDiskReadWriter(fileName)
			if err != nil {
				return nil, err
			}
			mf.filesCreated = append(mf.filesCreated, fileName)
			return readWriter, nil
		}
		return &cache.SliceReadWriter{}, nil
	}
}

func (mf *ReadWriterMetaFactory) Cleanup() {
	var failedRemovals []string
	for _, filename := range mf.filesCreated {
		err := os.Remove(filename)
		if err != nil {
			log.Error("could not remove temp file %v: %v", filename, err)
			failedRemovals = append(failedRemovals, filename)
		}
	}
	mf.filesCreated = failedRemovals
}

func makeFileName(layer uint) string {
	return fmt.Sprintf("poet_layercache_%d.bin", layer)
}
