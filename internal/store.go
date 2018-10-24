package internal

import (
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"os"
)

// todo: add buffered reader and writter wrapper

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

// Returns true iff node's label is in the store
func (d *KVFileStore) IsLabelInStore(id Identifier) (bool, error) {

	idx, err := d.calcFileIndex(id)
	if err != nil {
		return false, err
	}

	stats, err := d.file.Stat()
	if err != nil {
		return false, err
	}

	fileSize := uint64(stats.Size())

	return idx < fileSize, nil
}

// Returns the label of node id or error if it is not in the store
func (d *KVFileStore) Read(id Identifier) (shared.Label, error) {

	var label shared.Label

	idx, err := d.calcFileIndex(id)
	if err != nil {
		return label, err
	}

	_, err = d.file.Seek(int64(idx), 0)
	if err != nil {
		return label, err
	}

	// create a slice from the label array
	lSlice := label[:]

	//buff := make([]byte, shared.WB)

	n, err := d.file.Read(lSlice)
	if err != nil {
		return label, err
	}

	if n == 0 {
		return label, errors.New("label for id is ont in store")
	}

	return label, nil
}

func (d *KVFileStore) Write(id Identifier, l shared.Label) error {
	idx, err := d.calcFileIndex(id)
	if err != nil {
		return err
	}

	_, err = d.file.Seek(int64(idx), 0)
	if err != nil {
		return err
	}

	_, err = d.file.Write(l[:])
	return err
}

func (d *KVFileStore) calcFileIndex(id Identifier) (uint64, error) {
	s := d.subtreeSize(id)
	s1, err := d.leftSiblingsSubtreeSize(id)
	if err != nil {
		return 0, err
	}
	return s + s1, nil
}

// Returns the size of the subtree rooted at node id
func (d *KVFileStore) subtreeSize(id Identifier) uint64 {
	// node depth is the number of bits in its id
	depth := uint(len(id))
	height := d.n - depth
	return uint64(math.Pow(2, float64(height+1)) - 1)
}

// Returns the size of the subtrees rooted at left siblings on the path
// from node id to the root node
func (d *KVFileStore) leftSiblingsSubtreeSize(id Identifier) (uint64, error) {
	bs, err := d.f.NewBinaryString(string(id))
	if err != nil {
		return 0, err
	}

	siblings, err := bs.GetBNSiblings(true)
	if err != nil {
		return 0, err
	}
	var res uint64

	for _, s := range siblings {
		res += d.subtreeSize(Identifier(s.GetStringValue()))
	}

	return res, nil
}
