package service

import (
	"context"
	"errors"
)

var ErrChallengeInvalid = errors.New("challenge is invalid")

//go:generate mockgen -package mocks -destination mocks/challenge_verifier.go . Verifier

type Verifier interface {
	Verify(ctx context.Context, challenge, nodeID []byte, nonce uint64) error
}

type ChallengeVerifier struct{}

func (c *ChallengeVerifier) Verify(ctx context.Context, challenge, nodeID []byte, nonce uint64) error {
	// TODO(poszu): implement
	return nil
}
