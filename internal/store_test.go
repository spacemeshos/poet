package internal

import (
	"bytes"
	"crypto/rand"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

// helper method, writes to store and verify
// that value was written by reading it
func writeAndVerify(t *testing.T, s IKvStore, id Identifier, l shared.Label) {

	err := s.Write(id, l)
	assert.NoError(t, err)

	res, err := s.IsLabelInStore(id)
	assert.NoError(t, err)
	assert.True(t, res)

	l1, err := s.Read(id)
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(l[:], l1[:]))
}

func genRndLabel(t *testing.T) shared.Label {
	var l shared.Label
	_, err := rand.Read(l[:])
	assert.NoError(t, err)
	return l
}

func TestWrites(t *testing.T) {

	const n = 4

	s, err := NewKvFileStore("/Users/avive/dev/store.bin", n)
	assert.NoError(t, err)

	err = s.Reset()
	assert.NoError(t, err)

	writeAndVerify(t, s, "0000", genRndLabel(t))
	writeAndVerify(t, s, "0001", genRndLabel(t))
	writeAndVerify(t, s, "000", genRndLabel(t))

	writeAndVerify(t, s, "0010", genRndLabel(t))
	writeAndVerify(t, s, "0011", genRndLabel(t))
	writeAndVerify(t, s, "001", genRndLabel(t))

	writeAndVerify(t, s, "00", genRndLabel(t))

	writeAndVerify(t, s, "0100", genRndLabel(t))
	writeAndVerify(t, s, "0101", genRndLabel(t))
	writeAndVerify(t, s, "010", genRndLabel(t))

	writeAndVerify(t, s, "0110", genRndLabel(t))
	writeAndVerify(t, s, "0111", genRndLabel(t))
	writeAndVerify(t, s, "011", genRndLabel(t))

	writeAndVerify(t, s, "01", genRndLabel(t))

	writeAndVerify(t, s, "0", genRndLabel(t))

	// test for non-existing value
	res, err := s.IsLabelInStore(Identifier("1111"))
	assert.NoError(t, err)
	assert.False(t, res)

	err = s.Close()
	assert.NoError(t, err)
}

func TestDAG(t *testing.T) {

	const n = 2

	s, err := NewKvFileStore("/Users/avive/dev/store.bin", n)
	assert.NoError(t, err)

	err = s.Reset()
	assert.NoError(t, err)

	writeAndVerify(t, s, "00", genRndLabel(t))
	writeAndVerify(t, s, "01", genRndLabel(t))
	writeAndVerify(t, s, "0", genRndLabel(t))

	writeAndVerify(t, s, "10", genRndLabel(t))
	writeAndVerify(t, s, "11", genRndLabel(t))
	writeAndVerify(t, s, "1", genRndLabel(t))

	writeAndVerify(t, s, "", genRndLabel(t))

	assert.Equal(t, uint64(7), s.Size()/shared.WB)

	err = s.Close()
	assert.NoError(t, err)
}
