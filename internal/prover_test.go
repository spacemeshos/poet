package internal

import (
	"bytes"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
const x = "this is a commitment"
const n = 25
Map size:  67108863 ~20GB
Computed root label: 68b4c66918faa1a6538920944f13957354910f741a87236ea4905f2a50314c10
PASS: TestProverBasic (1034.77s)
*/

func TestProverBasic(t *testing.T) {

	const x = "this is a commitment"
	const n = 3

	p, err := NewProver([]byte(x), n)
	assert.NoError(t, err)

	p.ComputeDag(func(phi shared.Label, err error) {
		fmt.Printf("Root label: %x\n", phi)
		assert.NoError(t, err)

		// test root label computation from parents
		leftLabel, ok := p.GetLabel("0")
		assert.True(t, ok)
		rightLabel, ok := p.GetLabel("1")
		assert.True(t, ok)

		data := append([]byte(""), leftLabel[:]...)
		data = append(data, rightLabel[:]...)

		l := p.GetHashFunction().Hash(data)
		assert.True(t, bytes.Compare(phi[:], l[:]) == 0)
	})
}
