package api_test

import (
	"math/rand"
	"testing"

	"github.com/spacemeshos/ed25519"
	sharedPost "github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"

	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signing"
)

func TestOneOfMustBeSet(t *testing.T) {
	request := rpcapi.SubmitRequest{}
	_, err := api.FromSubmitRequest(&request)
	require.ErrorIs(t, err, api.ErrOneOfNotSet)
}

func TestSignatureVerification(t *testing.T) {
	request := rpcapi.SubmitRequest{
		Data: &rpcapi.SubmitRequest_Data{
			Prev: &rpcapi.SubmitRequest_Data_PrevAtxId{
				PrevAtxId: []byte{},
			},
		},
		Signature: make([]byte, ed25519.SignatureSize),
	}
	_, err := api.FromSubmitRequest(&request)
	require.ErrorIs(t, err, signing.ErrSignatureInvalid)
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()
	result := make([]byte, size)
	n, err := rand.Read(result)
	require.NoError(t, err)
	require.Equal(t, n, size)
	return result
}

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

func TestParsingRoundTrip(t *testing.T) {
	t.Parallel()

	testRoundTrip := func(t *testing.T, challenge shared.Challenge) {
		pubKey, privKey, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		signer := signer{
			privKey: privKey,
			pubKey:  pubKey,
		}
		signedData, err := signing.Sign(challenge, &signer)
		require.NoError(t, err)
		require.EqualValues(t, signedData.PubKey(), pubKey)

		grpcRequest, err := api.IntoSubmitRequest(signedData)
		require.NoError(t, err)

		parsedSignedData, err := api.FromSubmitRequest(grpcRequest)
		require.NoError(t, err)
		require.Equal(t, signedData, parsedSignedData)
		require.Equal(t, signedData.Data(), parsedSignedData.Data())
		require.Equal(t, signedData.PubKey(), parsedSignedData.PubKey())
		require.Equal(t, signedData.Signature(), parsedSignedData.Signature())
	}

	t.Run("Initial ATX case", func(t *testing.T) {
		t.Parallel()
		challenge := shared.Challenge{
			PositioningAtxId: randomBytes(t, 32),
			PubLayerId:       shared.LayerID(rand.Uint32()),
			InitialPost: &shared.InitialPost{
				Proof: sharedPost.Proof{
					Nonce:   rand.Uint32(),
					Indices: randomBytes(t, 32),
				},
				Metadata: sharedPost.ProofMetadata{
					Commitment:    randomBytes(t, 32),
					Challenge:     randomBytes(t, 32),
					NumUnits:      rand.Uint32(),
					BitsPerLabel:  uint8(rand.Uint32()),
					LabelsPerUnit: rand.Uint64(),
					K1:            rand.Uint32(),
					K2:            rand.Uint32(),
				},
			},
			PreviousATXId: nil,
		}
		testRoundTrip(t, challenge)
	})

	t.Run("Previous ATX case", func(t *testing.T) {
		t.Parallel()
		challenge := shared.Challenge{
			PositioningAtxId: randomBytes(t, 32),
			PubLayerId:       shared.LayerID(rand.Uint32()),
			InitialPost:      nil,
			PreviousATXId:    randomBytes(t, 32),
		}
		testRoundTrip(t, challenge)
	})
}
