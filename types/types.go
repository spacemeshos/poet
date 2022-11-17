package types

import (
	"context"
)

//go:generate mockgen -package mocks -destination mocks/challenge_verifier.go . ChallengeVerifier

type ChallengeVerifier interface {
	Verify(ctx context.Context, challenge []byte, signature []byte) (hash []byte, err error)
}
