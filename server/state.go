package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/state"
)

const stateFilename = "state.bin"

// Environment variable name for the private key.
// Should be set to a base64-encoded ed25519 private key.
const keyEnvVar = "POET_PRIVATE_KEY"

// The server state is persisted to disk.
type serverState struct {
	PrivKey []byte
}

func loadState(ctx context.Context, datadir, envKey string) (*serverState, error) {
	s := &serverState{}
	log := logging.FromContext(ctx)
	var key []byte
	if envKey != "" {
		var err error
		key, err = base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return nil, fmt.Errorf("decoding private key: %w", err)
		}
		if len(key) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid private key length: %d", len(key))
		}
		log.Info("loaded private key from environment")
	}

	err := state.Load(filepath.Join(datadir, stateFilename), s)
	switch {
	case errors.Is(err, os.ErrNotExist) && key == nil:
		pubKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("generating key: %w", err)
		}
		s.PrivKey = privateKey

		log.Info("generated new keys", zap.Binary("public key", pubKey))
	case errors.Is(err, os.ErrNotExist) && key != nil:
		s.PrivKey = key
	case err == nil && key != nil && !bytes.Equal(key, s.PrivKey):
		return nil, fmt.Errorf("private key mismatch. env: %s != %s: %s", key, stateFilename, s.PrivKey)
	case err != nil:
		return nil, fmt.Errorf("loading state: %w", err)
	}
	return s, nil
}

func saveState(datadir string, s *serverState) error {
	return state.Persist(filepath.Join(datadir, stateFilename), s)
}
