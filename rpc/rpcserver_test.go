package rpc

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/config"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
)

func Test_Submit_DoesNotPanicOnMissingPubKey(t *testing.T) {
	// Arrange
	cfg := config.DefaultConfig()
	sv := NewServer(nil, nil, *cfg)

	// Act
	in := &api.SubmitRequest{}
	out := &api.SubmitResponse{}
	var err error

	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
}

func Test_Submit_DoesNotPanicOnMissingSignature(t *testing.T) {
	// Arrange
	cfg := config.DefaultConfig()
	sv := NewServer(nil, nil, *cfg)
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Act
	in := &api.SubmitRequest{
		Pubkey: pub,
	}
	out := &api.SubmitResponse{}

	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
}
