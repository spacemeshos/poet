package internal

import (
	"fmt"
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

func TestProtocol(t *testing.T) {

	const x = "this is a commitment"
	const n = 9

	p, err := NewProver([]byte(x), n)
	assert.NoError(t, err)

	p.ComputeDag(func(phi Label, err error) {
		fmt.Printf("Root label: %x\n", phi)
		assert.NoError(t, err)

		//v, err := NewVerifier([]byte(x), n)
		//assert.NoError(t, err)

		//c, err := v.CreteNipChallenge(phi[:])
		//assert.NoError(t, err)

	})

}
