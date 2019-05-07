package prover

import (
	"bufio"
	"github.com/spacemeshos/merkle-tree"
	"io"
	"os"
)

const OwnerReadWrite = 0600

func NewDiskReadWriter(filename string) *DiskReadWriter {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, OwnerReadWrite)
	if err != nil {
		log.Error("failed to open file for disk read-writer: %v", err)
		panic(err) // TODO(noamnelke): support returning an error
	}
	return &DiskReadWriter{
		f: f,
		b: bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f)),
	}
}

type DiskReadWriter struct {
	f *os.File
	b *bufio.ReadWriter
}

func (rw *DiskReadWriter) Seek(index uint64) error {
	_, err := rw.f.Seek(int64(index*merkle.NodeSize), io.SeekStart)
	if err != nil {
		log.Error("failed to seek in disk reader: %v", err)
		return err
	}
	rw.b.Reader.Reset(rw.f)
	return err
}

func (rw *DiskReadWriter) ReadNext() ([]byte, error) {
	ret := make([]byte, merkle.NodeSize)
	_, err := rw.b.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (rw *DiskReadWriter) Width() uint64 {
	err := rw.b.Flush() // TODO(noamnelke): add flush method to interface and call when switching to read mode
	if err != nil {
		log.Error("failed to flush disk writer: %v", err)
		return 0
	}
	info, err := rw.f.Stat()
	if err != nil {
		log.Error("failed to get stats for disk reader: %v", err)
		return 0
	}
	return uint64(info.Size()) / merkle.NodeSize
}

func (rw *DiskReadWriter) Append(p []byte) (n int, err error) {
	n, err = rw.b.Write(p)
	return
}
