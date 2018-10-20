package internal

import (
	"encoding/hex"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNipChallenge(t *testing.T) {

	const x = "this is a commitment"
	const phi = "68b4c66918faa1a6538920944f13957354910f741a87236ea4905f2a50314c10"
	const n = 25

	v, err := NewVerifier([]byte(x), n)
	assert.NoError(t, err)

	phiData, err := hex.DecodeString(phi)
	assert.NoError(t, err)

	c, err := v.CreteNipChallenge(phiData)
	assert.NoError(t, err)

	assert.Equal(t, shared.T, len(c.Data), "Expected t identifiers in challenge")

	for _, id := range c.Data {
		assert.Equal(t, n, len(id), "Unexpected identifier width")
		println(id)
	}
}

func TestRndChallenge(t *testing.T) {
	const x = "this is a commitment"
	const n = 63
	v, err := NewVerifier([]byte(x), n)
	assert.NoError(t, err)

	c, err := v.CreteRndChallenge()
	assert.NoError(t, err)
	assert.Equal(t, shared.T, len(c.Data), "Expected t identifiers in challenge")

	for _, id := range c.Data {
		assert.Equal(t, n, len(id), "Unexpected identifier width")
		println(id)
	}
}
