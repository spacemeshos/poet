package types

import (
	"context"
)

//go:generate mockgen -package mocks -destination mocks/challenge_verifier.go . ChallengeVerifier

type ChallengeVerificationResult struct {
	Hash   []byte
	NodeId []byte
}

type ChallengeVerifier interface {
	Verify(ctx context.Context, challenge, signature []byte) (*ChallengeVerificationResult, error)
}
