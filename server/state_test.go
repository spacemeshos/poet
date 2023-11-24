package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadState(t *testing.T) {
	_, key, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	keyb64 := base64.StdEncoding.EncodeToString(key)

	t.Run("generate new key", func(t *testing.T) {
		s, err := loadState(context.Background(), t.TempDir(), "")
		require.NoError(t, err)
		require.NotNil(t, s.PrivKey)
	})
	t.Run("use key from ENV", func(t *testing.T) {
		s, err := loadState(context.Background(), t.TempDir(), keyb64)
		require.NoError(t, err)
		require.Equal(t, []byte(key), s.PrivKey)
	})
	t.Run("key must be 64B", func(t *testing.T) {
		_, err := loadState(context.Background(), t.TempDir(), "VGVzdA==")
		require.Error(t, err)
	})
	t.Run("key must be base64", func(t *testing.T) {
		_, err := loadState(context.Background(), t.TempDir(), "not b64")
		require.Error(t, err)
	})
	t.Run("detect mismatch between persisted key and env", func(t *testing.T) {
		dir := t.TempDir()
		s, err := loadState(context.Background(), dir, "")
		require.NoError(t, err)
		require.NoError(t, saveState(dir, s))

		// Set env to different key
		_, err = loadState(context.Background(), dir, keyb64)
		require.Error(t, err)
	})
	t.Run("persisting key", func(t *testing.T) {
		dir := t.TempDir()
		s, err := loadState(context.Background(), dir, "")
		require.NoError(t, err)
		require.NoError(t, saveState(dir, s))

		s2, err := loadState(context.Background(), dir, "")
		require.NoError(t, err)
		require.Equal(t, s.PrivKey, s2.PrivKey)
	})
}
