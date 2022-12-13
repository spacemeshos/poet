package service

import (
	"crypto/ed25519"
	"fmt"
	"path/filepath"
)

const serviceStateFileBaseName = "state.bin"

type serviceState struct {
	PrivKey []byte
}

func newServiceState() *serviceState {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(fmt.Errorf("failed to generate key: %v", err))
	}

	return &serviceState{PrivKey: priv}
}

func (s *serviceState) save(datadir string) error {
	return persist(filepath.Join(datadir, serviceStateFileBaseName), s)
}

func loadServiceState(datadir string) (*serviceState, error) {
	filename := filepath.Join(datadir, serviceStateFileBaseName)
	v := &serviceState{}

	if err := load(filename, v); err != nil {
		return nil, err
	}

	return v, nil
}
