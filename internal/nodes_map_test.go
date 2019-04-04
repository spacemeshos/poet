package internal

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNodesMap(t *testing.T) {
	f := NewSMBinaryStringFactory()

	const d = "0000010001011010010001100111000010010101101111111111101110000110"

	b, err := f.NewBinaryString(d)
	assert.NoError(t, err)

	m := NewNodesMap()

	_, ok := m.Get(b)
	assert.False(t, ok)

	label, _ := hex.DecodeString("68b4c66918faa1a6538920944f13957354910f741a87236ea4905f2a50314c10")
	m.Put(b, label)

	label1, ok := m.Get(b)
	assert.True(t, ok)

	assert.EqualValues(t, label, label1)

	b1, err := f.NewBinaryString(d)
	assert.NoError(t, err)

	label2, ok := m.Get(b1)
	assert.True(t, ok)

	assert.EqualValues(t, label1, label2)
}
