package verifier

import (
	"encoding/hex"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNipChallenge(t *testing.T) {
	x := []byte("this is a commitment")

	const n = 25

	data, _ := hex.DecodeString("68b4c66918faa1a6538920944f13957354910f741a87236ea4905f2a50314c10")
	var phi shared.Label
	copy(phi[:], data)

	v, err := New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreateNipChallenge(phi)
	assert.NoError(t, err)

	assert.Equal(t, shared.T, len(c.Data), "Expected t identifiers in challenge")

	for _, id := range c.Data {
		assert.Equal(t, n, len(id), "Unexpected identifier width")
		println(id)
	}
}

func TestRndChallenge(t *testing.T) {
	x := []byte("this is a commitment")
	const n = 29

	v, err := New(x, n, shared.NewScryptHashFunc(x))
	assert.NoError(t, err)

	c, err := v.CreateRndChallenge()
	assert.NoError(t, err)
	assert.Equal(t, shared.T, len(c.Data), "Expected t identifiers in challenge")

	for _, id := range c.Data {
		assert.Equal(t, n, len(id), "Unexpected identifier width")
		println(id)
	}
}
