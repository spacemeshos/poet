package internal

import (
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"os"
	"sync"
)

type WriteData struct {
	id shared.Identifier
	l  shared.Label
}

type WriteChan chan *WriteData

type IKvStore interface {
	Read(id Identifier) (shared.Label, error)
	Write(id Identifier, l shared.Label)
	IsLabelInStore(id Identifier) (bool, error)
	Reset() error
	Delete() error
	Size() uint64
	Finalize()    // finalize writing w/o closing the file
	Close() error // finalize and close

	//GetWriteChan() WriteChan
}

type KVFileStore struct {
	fileName string
	file     *os.File
	n        uint // 1 <= n < 64
	f        BinaryStringFactory
	bw       *Writer
	c        uint64 // num of labels written to store in this session
	wc       WriteChan
	wg       sync.WaitGroup
	once     sync.Once
}

const buffSizeBytes = 1024 * 1024 * 1024

// Create a new prover with commitment X and 1 <= n < 64
// n specifies the leafs height from the root and the number of bits in leaf ids
// buffSize - memory buffer size in bytes
func NewKvFileStore(fileName string, n uint) (IKvStore, error) {

	res := &KVFileStore{
		fileName: fileName,
		n:        n,
		f:        NewSMBinaryStringFactory(),
		wc:       make(WriteChan, 100),
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

	// todo: compare pref w/o buffers
	d.bw = NewWriterSize(f, buffSizeBytes)

	d.wg.Add(1)

	go d.beginEventProcessing()

	return nil
}

func (d *KVFileStore) beginEventProcessing() {
	defer d.wg.Done()
	for dataItem := range d.wc {
		d.c += 1
		_, err := d.bw.Write(dataItem.l)
		if err != nil {
			panic(err)
		}
	}
}

func (d *KVFileStore) Write(id Identifier, l shared.Label) {
	d.wc <- &WriteData{id, l}
}

// Removes all data from the file
func (d *KVFileStore) Reset() error {
	d.bw.Flush()
	d.c = 0
	return d.file.Truncate(0)
}

func (d *KVFileStore) Finalize() {

	d.once.Do(func() { close(d.wc) })

	// wait for all buffered writes to be added to the buffer
	d.wg.Wait()

	// flush buffer to file
	d.bw.Flush()
}

func (d *KVFileStore) Close() error {
	d.Finalize()
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

	return uint64(stats.Size()) + uint64(d.bw.Buffered())
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

	if d.bw.Buffered() > 0 && idx < (d.c*shared.WB) {
		// label is in file or in the buffer
		return true, nil
	}

	fileSize := uint64(stats.Size())
	return idx < fileSize, nil
}

// Read label value from the store
// Returns the label of node id or error if it is not in the store
func (d *KVFileStore) Read(id Identifier) (shared.Label, error) {

	label := make(shared.Label, shared.WB)

	// total # of labels written - # of buffered labels == idx of label at buff start
	// say 4 labels were written, and Buffered() is 64 bytes. 2 last labels
	// are in buffer and the index of the label at buff start is 2.
	idAtBuffStart := d.c - uint64(d.bw.Buffered()/shared.WB)

	// label file index
	idx, err := d.calcFileIndex(id)
	if err != nil {
		return label, err
	}

	idxBuffStart := idAtBuffStart * shared.WB

	if idx >= idxBuffStart {
		// label is in buffer - we need to flush it to file before reading

		// todo: find best way to just read the data from the buffer w/o flushing
		// this might be a significant optimization - more profiling needed

		d.bw.Flush()
	}

	n, err := d.file.ReadAt(label, int64(idx))
	if err != nil {
		return label, err
	}

	if n == 0 {
		return label, errors.New("label for id is not in store")
	}

	return label, nil
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
