package internal

import (
	"bytes"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProverBasic(t *testing.T) {
	x := []byte("this is a commitment")
	const n = 9

	p, err := NewProver(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	phi, err := p.ComputeDag()

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
	assert.True(t, bytes.Equal(phi[:], l[:]))
}

func BenchmarkProver(t *testing.B) {
	x := []byte("this is a commitment")
	const n = 15
	p, err := NewProver(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)
	_, _ = p.ComputeDag()
}
