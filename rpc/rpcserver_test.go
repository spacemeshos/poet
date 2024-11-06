package rpc_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/rpc"
)

func Test_Submit_DoesNotPanicOnMissingChallenge(t *testing.T) {
	// Arrange
	sv := rpc.NewServer(nil, 0, 0)

	// Act
	in := &api.SubmitRequest{}
	out := &api.SubmitResponse{}

	var err error
	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "invalid challenge")
}

func Test_Submit_DoesNotPanicOnMissingPubKey(t *testing.T) {
	// Arrange
	sv := rpc.NewServer(nil, 0, 0)

	challenge := make([]byte, 32)
	n, err := rand.Read(challenge)
	require.NoError(t, err)
	require.Equal(t, 32, n)

	// Act
	in := &api.SubmitRequest{
		Challenge: challenge,
	}
	out := &api.SubmitResponse{}

	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "invalid public key")
}

func Test_Submit_DoesNotPanicOnMissingSignature(t *testing.T) {
	// Arrange
	sv := rpc.NewServer(nil, 0, 0)
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	challenge := make([]byte, 32)
	n, err := rand.Read(challenge)
	require.NoError(t, err)
	require.Equal(t, 32, n)

	// Act
	in := &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pub,
	}
	out := &api.SubmitResponse{}

	require.NotPanics(t, func() { out, err = sv.Submit(context.Background(), in) })

	// Assert
	require.Nil(t, out)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "invalid signature")
}
