package signing_test

import (
	"testing"

	"github.com/spacemeshos/ed25519"
	"github.com/spacemeshos/go-scale"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/signing"
)

type signer struct {
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

func (s *signer) Sign(data []byte) []byte {
	return ed25519.Sign2(s.privKey, data)
}

func (s *signer) PublicKey() []byte {
	return s.pubKey
}

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
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(err)
	signer := signer{
		privKey: privKey,
		pubKey:  pubKey,
	}

	// Sign
	require.NoError(err)
	signed, err := signing.Sign(data, &signer)
	require.NoError(err)
	require.EqualValues(data, *signed.Data())

	// Create Signed from a signed data
	signed2, err := signing.NewFromScaleEncodable(*signed.Data(), signed.Signature())
	require.NoError(err)
	require.EqualValues(signed2.Data(), signed.Data())
}

func TestInvalidSignature(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	data := Foo{s: "sign me"}

	_, err := signing.NewFromScaleEncodable(data, []byte{})
	require.ErrorIs(err, signing.ErrSignatureInvalid)
}
