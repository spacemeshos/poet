package internal

import (
	"bufio"
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"os"
)

type IKvStore interface {
	Read(id Identifier) (shared.Label, error)
	Write(id Identifier, l shared.Label) error
	IsLabelInStore(id Identifier) (bool, error)
	Reset() error
	Close() error
	Delete() error
	Size() uint64
}

type KVFileStore struct {
	fileName string
	file     *os.File
	n        uint // 1 <= n < 64
	f        BinaryStringFactory
	bw       *bufio.Writer
	dirty    bool
}

// Create a new prover with commitment X and 1 <= n < 64
// n specifies the leafs height from the root and the number of bits in leaf ids
// buffSize - memory buffer size in bytes
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

	f, err := os.OpenFile(d.fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	d.file = f

	// create buffer with default buf size
	// tood: compare pref w/o buffers
	d.bw = bufio.NewWriterSize(f, 4096 * 100000)

	return nil
}

// Removes all data from the file
func (d *KVFileStore) Reset() error {
	return d.file.Truncate(0)
}

func (d *KVFileStore) Close() error {
	if d.dirty { // must flush before reading
		d.bw.Flush()
		d.dirty = false
	}
	return d.file.Close()
}

func (d *KVFileStore) Delete() error {
	return os.Remove(d.fileName)
}

func (d *KVFileStore) Size() uint64 {
	stats, err := d.file.Stat()
	if err != nil {
		println(err)
	}
	return uint64(stats.Size())
}

// Returns true iff node's label is in the store
func (d *KVFileStore) IsLabelInStore(id Identifier) (bool, error) {

	if d.dirty { // must flush before reading
		d.bw.Flush()
		d.dirty = false
	}

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

	if d.dirty { // must flush before reading
		d.bw.Flush()
		d.dirty = false
	}

	var label shared.Label

	idx, err := d.calcFileIndex(id)
	if err != nil {
		return label, err
	}

	// create a slice from the label array
	// lSlice := label[:]
	// buff := make([]byte, shared.WB)

	n, err := d.file.ReadAt(label[:], int64(idx))
	if err != nil {
		return label, err
	}

	if n == 0 {
		return label, errors.New("label for id is not in store")
	}

	return label, nil
}

func (d *KVFileStore) Write(id Identifier, l shared.Label) error {

	_, err := d.bw.Write(l[:])
	d.dirty = true
	return err

	// no need to calc index or to WriteAt because writes are sequential
	//

	/* no need to calc index or to WriteAt because writes are sequential
		idx, err := d.calcFileIndex(id)
		if err != nil {
			return err
		}

		//fmt.Printf("Writing %d bytes in offset %d\n", len(l[:]), idx)
		_, err = d.file.WriteAt(l[:], int64(idx))
		return err
	*/
}

// Returns the file offset for a node id
func (d *KVFileStore) calcFileIndex(id Identifier) (uint64, error) {
	s := d.subtreeSize(id)
	s1, err := d.leftSiblingsSubtreeSize(id)
	if err != nil {
		return 0, err
	}

	idx := s + s1 - 1
	offset := idx * shared.WB
	//fmt.Printf("Node id %s. Index: %d. Offset: %d\n", id, idx, offset)
	return offset, nil
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
