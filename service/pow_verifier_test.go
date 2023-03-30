package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
)

func TestVerify(t *testing.T) {
	t.Parallel()

	nodeID := make([]byte, 32)
	data := []byte("some data to submit to poet")
	challenge := []byte("challenge this")
	difficulty := uint(4)

	nonce, err := shared.SubmitPow(context.Background(), challenge, data, nodeID, difficulty)
	require.NoError(t, err)

	verifier := NewPowVerifier(NewPowParams(challenge, difficulty))
	require.NoError(t, verifier.Verify(data, nodeID, nonce))
	// Invalid nonce
	require.ErrorIs(t, verifier.Verify(data, nodeID, nonce-1), ErrInvalidPow)
	// Invalid nodeID
	require.ErrorIs(t, verifier.Verify(data, nodeID[1:], nonce), ErrInvalidPow)
}

func TestVerifyWithParams(t *testing.T) {
	t.Parallel()

	nodeID := make([]byte, 32)
	data := []byte("some data to submit to poet")

	verifiers := powVerifiers{
		previous: NewPowVerifier(NewPowParams([]byte("1"), 7)),
		current:  NewPowVerifier(NewPowParams([]byte("0"), 10)),
	}

	nonce, err := shared.SubmitPow(context.Background(), []byte("1"), data, nodeID, 7)
	require.NoError(t, err)

	err = verifiers.VerifyWithParams(data, nodeID, nonce, NewPowParams([]byte("0"), 7))
	require.ErrorIs(t, err, ErrInvalidPowParams)
	err = verifiers.VerifyWithParams(data, nodeID, nonce, NewPowParams([]byte("1"), 12))
	require.ErrorIs(t, err, ErrInvalidPowParams)

	err = verifiers.VerifyWithParams(data, nodeID, nonce, NewPowParams([]byte("1"), 7))
	require.NoError(t, err)
}

func TestGetPowParams(t *testing.T) {
	t.Parallel()
	params := NewPowParams([]byte("challenge this"), 7)
	require.Equal(t, params, NewPowVerifier(params).Params())
}
