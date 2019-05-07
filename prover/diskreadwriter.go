package prover

import (
	"bufio"
	"github.com/spacemeshos/merkle-tree"
	"io"
	"os"
)

const OwnerReadWrite = 0600

func NewDiskReadWriter(filename string) (*DiskReadWriter, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, OwnerReadWrite)
	if err != nil {
		log.Error("failed to open file for disk read-writer: %v", err)
		return nil, err
	}
	return &DiskReadWriter{
		f: f,
		b: bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f)),
	}, nil
}

type DiskReadWriter struct {
	f *os.File
	b *bufio.ReadWriter
}

func (rw *DiskReadWriter) Flush() error {
	err := rw.b.Flush()
	if err != nil {
		log.Error("failed to flush disk writer: %v", err)
		return err
	}
	return nil
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

func (rw *DiskReadWriter) Width() (uint64, error) {
	info, err := rw.f.Stat()
	if err != nil {
		log.Error("failed to get stats for disk reader: %v", err)
		return 0, err
	}
	return uint64(info.Size()) / merkle.NodeSize, nil
}

func (rw *DiskReadWriter) Append(p []byte) (n int, err error) {
	n, err = rw.b.Write(p)
	return
}
