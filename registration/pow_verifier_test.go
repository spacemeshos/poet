package registration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
)

func TestVerify(t *testing.T) {
	t.Parallel()

	nodeID := make([]byte, 32)
	poetChallenge := []byte("some data to submit to poet")
	challenge := []byte("challenge this")
	difficulty := uint(6)

	nonce, err := shared.FindSubmitPowNonce(context.Background(), challenge, poetChallenge, nodeID, difficulty)
	require.NoError(t, err)
	require.EqualValues(t, 143, nonce)

	verifier := NewPowVerifier(NewPowParams(challenge, difficulty))
	require.NoError(t, verifier.Verify(poetChallenge, nodeID, nonce))
	// Invalid nonce
	require.ErrorIs(t, verifier.Verify(poetChallenge, nodeID, nonce-1), ErrInvalidPow)
	// Invalid nodeID
	require.ErrorIs(t, verifier.Verify(poetChallenge, nodeID[1:], nonce), ErrInvalidPow)
}

func TestVerifyWithParams(t *testing.T) {
	t.Parallel()

	nodeID := make([]byte, 32)
	poetChallenge := []byte("some data to submit to poet")

	verifiers := powVerifiers{
		previous: NewPowVerifier(NewPowParams([]byte("1"), 7)),
		current:  NewPowVerifier(NewPowParams([]byte("0"), 10)),
	}

	nonce, err := shared.FindSubmitPowNonce(context.Background(), []byte("1"), poetChallenge, nodeID, 7)
	require.NoError(t, err)

	err = verifiers.VerifyWithParams(poetChallenge, nodeID, nonce, NewPowParams([]byte("0"), 7))
	require.ErrorIs(t, err, ErrInvalidPowParams)
	err = verifiers.VerifyWithParams(poetChallenge, nodeID, nonce, NewPowParams([]byte("1"), 12))
	require.ErrorIs(t, err, ErrInvalidPowParams)

	err = verifiers.VerifyWithParams(poetChallenge, nodeID, nonce, NewPowParams([]byte("1"), 7))
	require.NoError(t, err)
}

func TestGetPowParams(t *testing.T) {
	t.Parallel()
	params := NewPowParams([]byte("challenge this"), 7)
	require.Equal(t, params, NewPowVerifier(params).Params())
}
