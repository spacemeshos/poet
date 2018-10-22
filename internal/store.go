package internal

import (
	"github.com/spacemeshos/poet-ref/shared"
	"os"
)

type KVFileStore struct {
	fileName string
	file     *os.File
	n        uint // 1 <= n < 64
}

func (d *KVFileStore) init() error {
	f, err := os.Create(d.fileName)
	if err != nil {
		d.file = f
	}
	return err
}

// You can store labels in the order in which they are computed,
// and given a label reconstruct its index easily:
// idx = sum of sizes of the subtrees under the left-siblings on path to root +
// node’s own subtree. The size of a subtree under a node is simply 2^{height+1}-1
// plus id as int value?

func (d *KVFileStore) read(id Identifier) (shared.Label, error) {
	return shared.Label{}, nil
}

func (d *KVFileStore) write(id Identifier, l shared.Label) error {
	return nil
}

func calcFileIndex(id Identifier) uint64 {
	return 0
}
