package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProverBasic(t *testing.T) {

	const x = "this is a commitment"
	const n = 23

	p, err := NewProver([]byte(x), n)
	assert.NoError(t, err)

	p.CreateProof(func (phi Label, err error) {
		fmt.Printf("Root label: %x\n", phi)
		assert.NoError(t, err)
	})
}
