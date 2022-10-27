package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/poet/signing"
	"github.com/stretchr/testify/require"
)

type Foo struct {
	s string
}

func (f *Foo) EncodeScale(enc *scale.Encoder) (int, error) {
	return scale.EncodeString(enc, f.s)
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	data := Foo{s: "sign me"}

	// Sign
	pubKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(err)
	signed, err := signing.Sign(data, privateKey, pubKey)
	require.NoError(err)
	require.EqualValues(data, *signed.Data())

	// Create Signed from a signed data
	signed2, err := signing.NewFromScaleEncodable(*signed.Data(), signed.Signature(), signed.PubKey())
	require.NoError(err)
	require.EqualValues(signed2.Data(), signed.Data())
}

func TestInvalidSignature(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	data := Foo{s: "sign me"}

	pubKey, _, err := ed25519.GenerateKey(nil)
	require.NoError(err)

	_, err = signing.NewFromScaleEncodable(data, []byte{}, pubKey)
	require.ErrorIs(err, signing.ErrSignatureInvalid)
}

func TestInvalidPubkey(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	data := Foo{s: "sign me"}

	_, err := signing.NewFromScaleEncodable(data, []byte{}, []byte{})
	require.ErrorIs(err, signing.ErrInvalidPubkeyLen)
}
