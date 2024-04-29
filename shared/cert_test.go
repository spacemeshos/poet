package shared

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_DecodeCert(t *testing.T) {
	encoded, err := base64.StdEncoding.DecodeString("gAECAwQFBgcICQoLDA0ODxABAgMEBQYHCAkKCwwNDg8QAXaB58o=")
	require.NoError(t, err)

	cert, err := DecodeCert(encoded)
	require.NoError(t, err)
	expId := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
	}
	require.Equal(t, expId, cert.Pubkey)

	exp, err := time.Parse(time.RFC3339, "1996-12-19T16:39:57-08:00")
	require.NoError(t, err)
	require.Equal(t, exp.UTC(), *cert.Expiration)
}

func createOpaqueCert(t *testing.T, c Cert, priv ed25519.PrivateKey) *OpaqueCert {
	t.Helper()
	data, err := EncodeCert(&c)
	require.NoError(t, err)
	return &OpaqueCert{
		Data:      data,
		Signature: ed25519.Sign(priv, data),
	}
}

func Test_VerifyCertificate(t *testing.T) {
	pub, private, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	nodeID := []byte("nodeID00nodeID00nodeID00nodeID00")
	t.Run("valid certificate w/o expiration", func(t *testing.T) {
		cert := createOpaqueCert(t, Cert{Pubkey: nodeID}, private)
		c, err := VerifyCertificate(cert, pub, nodeID)
		require.NoError(t, err)
		require.Nil(t, c.Expiration)
		require.Equal(t, nodeID, c.Pubkey)
	})

	t.Run("valid certificate (unexpired)", func(t *testing.T) {
		expiration := time.Now().Add(time.Hour)
		cert := createOpaqueCert(t, Cert{Pubkey: nodeID, Expiration: &expiration}, private)
		c, err := VerifyCertificate(cert, pub, nodeID)
		require.NoError(t, err)
		require.Equal(t, expiration.Unix(), c.Expiration.Unix())
		require.Equal(t, nodeID, c.Pubkey)
	})
	t.Run("valid certificate (expired)", func(t *testing.T) {
		expiration := time.Now().Add(-time.Hour)
		cert := createOpaqueCert(t, Cert{Pubkey: nodeID, Expiration: &expiration}, private)
		_, err := VerifyCertificate(cert, pub, nodeID)
		require.ErrorIs(t, err, ErrCertExpired)
	})
	t.Run("valid certificate (expired - time is zero)", func(t *testing.T) {
		cert := createOpaqueCert(t, Cert{Pubkey: nodeID, Expiration: &time.Time{}}, private)
		_, err := VerifyCertificate(cert, pub, nodeID)
		require.Error(t, err)
	})

	t.Run("certificate for different node ID", func(t *testing.T) {
		cert := createOpaqueCert(t, Cert{Pubkey: []byte("wrong node ID")}, private)
		_, err := VerifyCertificate(cert, pub, nodeID)
		require.Error(t, err)
	})

	t.Run("invalid certificate", func(t *testing.T) {
		data := []byte{1, 2, 3, 4}
		cert := &OpaqueCert{
			Data:      data,
			Signature: ed25519.Sign(private, data),
		}
		_, err := VerifyCertificate(cert, pub, nodeID)
		require.Error(t, err)
	})
	t.Run("invalid certificate (wrong signature)", func(t *testing.T) {
		_, wrongPrivate, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		cert := createOpaqueCert(t, Cert{Pubkey: nodeID}, wrongPrivate)
		_, err = VerifyCertificate(cert, pub, nodeID)
		require.Error(t, err)
	})
}
