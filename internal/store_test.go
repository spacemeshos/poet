package internal

import (
	"crypto/rand"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// helper method, writes to store and verify
// that value was written by reading it
func writeAndVerify(t *testing.T, s IKvStore, id Identifier, l shared.Label) {
	s.Write(id, l)
	s.Close()
}

func genRndLabel(t *testing.T) shared.Label {
	l := make(shared.Label, shared.WB)
	_, err := rand.Read(l[:])
	assert.NoError(t, err)
	return l
}

func TestWrites(t *testing.T) {

	const n = 4

	s, err := NewKvFileStore(filepath.Join(os.TempDir(), "store.bin"), n)
	assert.NoError(t, err)

	err = s.Reset()
	assert.NoError(t, err)

	s.Write("0000", genRndLabel(t))
	s.Write("0001", genRndLabel(t))
	s.Write("0011", genRndLabel(t))
	s.Write("001", genRndLabel(t))

	s.Write("00", genRndLabel(t))
	s.Write("0100", genRndLabel(t))
	s.Write("0101", genRndLabel(t))
	s.Write("010", genRndLabel(t))

	s.Write("0110", genRndLabel(t))
	s.Write("0111", genRndLabel(t))
	s.Write("011", genRndLabel(t))
	s.Write("01", genRndLabel(t))
	s.Write("0", genRndLabel(t))

	err = s.Finalize()
	assert.NoError(t, err)

	res, err := s.IsLabelInStore("0000")
	assert.NoError(t, err)
	assert.True(t, res)

	// test for non-existing value
	res, err = s.IsLabelInStore(Identifier("1111"))
	assert.NoError(t, err)
	assert.False(t, res)

	err = s.Close()
	assert.NoError(t, err)
}

func TestDAG(t *testing.T) {
	const n = 2

	s, err := NewKvFileStore(filepath.Join(os.TempDir(), "store.bin"), n)
	assert.NoError(t, err)

	err = s.Reset()
	assert.NoError(t, err)

	s.Write("01", genRndLabel(t))
	s.Write("0", genRndLabel(t))
	s.Write("10", genRndLabel(t))
	s.Write("11", genRndLabel(t))
	s.Write("", genRndLabel(t))

	err = s.Finalize()
	assert.NoError(t, err)

	assert.Equal(t, uint64(5), s.Size()/shared.WB)

	err = s.Close()
	assert.NoError(t, err)
}
