package service

import (
	"fmt"
	"path/filepath"

	"golang.org/x/crypto/ed25519"
)

func initialState() *serviceState {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(fmt.Errorf("failed to generate key: %v", err))
	}

	return &serviceState{PrivKey: priv}
}

func saveState(datadir string, privateKey ed25519.PrivateKey) error {
	filename := filepath.Join(datadir, serviceStateFileBaseName)
	return persist(filename, &serviceState{PrivKey: privateKey})
}

func state(datadir string) (*serviceState, error) {
	filename := filepath.Join(datadir, roundStateFileBaseName)
	v := &serviceState{}

	if err := load(filename, v); err != nil {
		return nil, err
	}

	return v, nil
}
