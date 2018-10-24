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
func writeAndVerify(t *testing.T, s IKvStore, id Identifier, l shared.Label)  {

	err := s.Write(id, l)
	assert.NoError(t, err)

	res, err := s.IsLabelInStore(id)
	assert.NoError(t, err)
	assert.True(t, res)

	l1, err := s.Read(id)
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(l[:], l1[:]))
}

func TestWrites(t *testing.T) {

	const n = 4

	var l, l1, l2 shared.Label
	_, err := rand.Read(l[:])
	assert.NoError(t, err)

	_, err = rand.Read(l1[:])
	assert.NoError(t, err)

	_, err = rand.Read(l2[:])
	assert.NoError(t, err)

	s, err := NewKvFileStore("/Users/avive/dev/store.bin", n)
	assert.NoError(t, err)

	s.Reset()

	writeAndVerify(t, s, "0000", l)
	writeAndVerify(t, s, "0001", l1)
	writeAndVerify(t, s, "000", l2)

	// test for non-existing value
	res, err := s.IsLabelInStore(Identifier("00"))
	assert.NoError(t, err)
	assert.False(t, res)

	err = s.Close()
	assert.NoError(t, err)

}


