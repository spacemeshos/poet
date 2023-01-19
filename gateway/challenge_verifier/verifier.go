package challenge_verifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/spacemeshos/sha256-simd"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
)

const MaxBackoff = time.Second * 30

var (
	ErrChallengeInvalid = errors.New("challenge is invalid")
	ErrCouldNotVerify   = errors.New("could not verify the challenge")
)

//go:generate mockgen -package mocks -destination mocks/challenge_verifier.go . Verifier

type Result struct {
	Hash   []byte
	NodeId []byte
}

type Verifier interface {
	Verify(ctx context.Context, challenge, signature []byte) (*Result, error)
}

// roundRobin gathers many verifiers.
// It tries to verify a challenge in a round robin fashion,
// retrying with the next verifier if the previous one was not
// able to complete verification.
type roundRobin struct {
	services    []Verifier
	lastUsedSvc int
}

func (a *roundRobin) Verify(ctx context.Context, challenge, signature []byte) (*Result, error) {
	for retries := 0; retries < len(a.services); retries++ {
		hash, err := a.services[a.lastUsedSvc].Verify(ctx, challenge, signature)
		if err == nil {
			return hash, nil
		}
		if errors.Is(err, ErrChallengeInvalid) {
			return nil, err
		}
		a.lastUsedSvc = (a.lastUsedSvc + 1) % len(a.services)
	}

	return nil, ErrCouldNotVerify
}

func NewRoundRobin(services []Verifier) Verifier {
	return &roundRobin{
		services: services,
	}
}

// caching implements caching layer on top of
// its ChallengeVerifier.
type caching struct {
	cache    *lru.Cache
	verifier Verifier
}

type challengeVerifierResult struct {
	*Result
	err error
}

func (a *caching) Verify(ctx context.Context, challenge, signature []byte) (*Result, error) {
	var challengeHash [sha256.Size]byte
	hasher := sha256.New()
	hasher.Write(challenge)
	hasher.Write(signature)
	hasher.Sum(challengeHash[:0])

	logger := logging.FromContext(ctx).With(zap.Binary("challenge", challengeHash[:]))
	if result, ok := a.cache.Get(challengeHash); ok {
		logger.Debug("retrieved challenge verifier result from the cache")
		// SAFETY: type assertion will never panic as we insert only `*challengeVerifierResult` values.
		result := result.(*challengeVerifierResult)
		return result.Result, result.err
	}

	result, err := a.verifier.Verify(ctx, challenge, signature)
	if err == nil || errors.Is(err, ErrChallengeInvalid) {
		a.cache.Add(challengeHash, &challengeVerifierResult{Result: result, err: err})
	}
	return result, err
}

func NewCaching(size int, verifier Verifier) (Verifier, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &caching{
		cache:    cache,
		verifier: verifier,
	}, nil
}

type retrying struct {
	backoffBase       time.Duration
	backoffMultiplier float64
	maxRetries        uint
	verifier          Verifier
}

func (v *retrying) Verify(ctx context.Context, challenge, signature []byte) (*Result, error) {
	logger := logging.FromContext(ctx)
	timer := time.NewTimer(0)
	<-timer.C
	delay := v.backoffBase

	for retry := uint(0); retry < v.maxRetries; retry++ {
		if hash, err := v.verifier.Verify(ctx, challenge, signature); err == nil {
			return hash, nil
		} else if errors.Is(err, ErrChallengeInvalid) {
			return nil, err
		}
		timer.Reset(delay)
		logger.Sugar().Infof("Retrying for %d time, waiting %v", retry+1, delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			logger.Debug("Retry interrupted", zap.Error(ctx.Err()))
			return nil, fmt.Errorf("%w: %v", ErrCouldNotVerify, ctx.Err())
		}
		delay = time.Duration(float64(delay) * v.backoffMultiplier)
		if delay > MaxBackoff {
			delay = MaxBackoff
		}
	}
	return nil, ErrCouldNotVerify
}

func NewRetrying(verifier Verifier, maxRetries uint, backoffBase time.Duration, backoffMultiplier float64) Verifier {
	return &retrying{
		maxRetries:        maxRetries,
		verifier:          verifier,
		backoffBase:       backoffBase,
		backoffMultiplier: backoffMultiplier,
	}
}
