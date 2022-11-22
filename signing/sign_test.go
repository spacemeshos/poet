package signing_test

import (
	"testing"

	"github.com/spacemeshos/ed25519"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/signing"
)

func TestCreatingSigned(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(err)

	data := []byte("data")
	signature := ed25519.Sign2(privKey, data)

	signed, err := signing.NewFromBytes(data, signature)
	require.NoError(err)
	require.EqualValues(data, *signed.Data())
}

func TestInvalidSignature(t *testing.T) {
	t.Parallel()
	_, err := signing.NewFromBytes([]byte("data"), []byte{})
	require.ErrorIs(t, err, signing.ErrSignatureInvalid)
}
