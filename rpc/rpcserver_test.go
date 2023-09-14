package rpc_test

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/rpc"
)

func Test_Submit_DoesNotPanicOnMissingPubKey(t *testing.T) {
	// Arrange
	sv := rpc.NewServer(nil, nil, 0, 0)

	// Act
	in := &api.SubmitRequest{}
	out := &api.SubmitResponse{}
	var err error

	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
	require.Equal(t, status.Code(err), codes.InvalidArgument)
	require.ErrorContains(t, err, "invalid public key")
}

func Test_Submit_DoesNotPanicOnMissingSignature(t *testing.T) {
	// Arrange
	sv := rpc.NewServer(nil, nil, 0, 0)
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
	require.Equal(t, status.Code(err), codes.InvalidArgument)
	require.ErrorContains(t, err, "invalid signature")
}
