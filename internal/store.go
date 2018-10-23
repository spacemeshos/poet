package internal

import (
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"os"
)

type IKvStore interface {
	Read(id Identifier) (shared.Label, error)
	Write(id Identifier, l shared.Label) error
}

type KVFileStore struct {
	fileName string
	file     *os.File
	n        uint // 1 <= n < 64
	f        BinaryStringFactory
}

// Create a new prover with commitment X and param 1 <= n <= 63
func NewKvFileStore(fileName string, n uint) (IKvStore, error) {

	res := &KVFileStore{
		fileName: fileName,
		n:        n,
		f:        NewSMBinaryStringFactory(),
	}

	err := res.init()

	return res, err
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

func (d *KVFileStore) Read(id Identifier) (shared.Label, error) {
	return shared.Label{}, nil
}

func (d *KVFileStore) Write(id Identifier, l shared.Label) error {
	return nil
}

func calcFileIndex(id Identifier) uint64 {
	return 0
}

// given a node id, return the size of the subtree rooted with it
func (d *KVFileStore) subtreeSize(id Identifier) uint64 {
	// node depth is the number of bits in its id
	depth := uint(len(id))
	height := d.n - depth
	return uint64(math.Pow(2, float64(height+1)) - 1)

}
