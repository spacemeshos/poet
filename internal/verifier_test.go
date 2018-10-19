package internal

import (
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNipChallenge(t *testing.T) {

	const x = "this is a commitment"
	const phi = "this is a mock root label"
	const n = 41
	v := NewVerifier([]byte(x), n)

	c, err := v.CreteNipChallenge([]byte(phi))

	assert.NoError(t, err)
	assert.Equal(t, shared.T, len(c.Data))

	for _, id := range c.Data {
		assert.Equal(t, n, len(id))
		println(id)
	}
}

func TestRndChallenge(t *testing.T) {
	const x = "this is a commitment"
	const n = 41
	v := NewVerifier([]byte(x), n)

	c, err := v.CreteRndChallenge()

	assert.NoError(t, err)
	assert.Equal(t, shared.T, len(c.Data))

	for _, id := range c.Data {
		assert.Equal(t, n, len(id))
		println(id)
	}
}

