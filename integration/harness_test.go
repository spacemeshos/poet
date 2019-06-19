package integration

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHarness(t *testing.T) {
	assert := require.New(t)

	cfg, err := DefaultConfig()
	assert.NoError(err)
	cfg.NodeAddress = "NO_BROADCAST"
	h, err := NewHarness(cfg)
	defer func() {
		err := h.TearDown()
		assert.NoError(err)
	}()
	assert.NoError(err)
	assert.NotNil(h)
}
