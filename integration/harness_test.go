package integration

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHarness(t *testing.T) {
	assert := require.New(t)

	h, err := NewHarness()
	defer func() {
		err := h.TearDown()
		assert.NoError(err)
	}()
	assert.NoError(err)
	assert.NotNil(h)
}
