package challenge_verifier

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/spacemeshos/smutil/log"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/types"
)

const MaxBackoff = time.Second * 30

// roundRobinChallengeVerifier gathers many verifiers.
// It tries to verify a challenge in a round robin fashion,
// retrying with the next verifier if the previous one was not
// able to complete verification.
type roundRobinChallengeVerifier struct {
	services    []types.ChallengeVerifier
	lastUsedSvc int
}

func (a *roundRobinChallengeVerifier) Verify(ctx context.Context, challenge, signature []byte) (*types.ChallengeVerificationResult, error) {
	for retries := 0; retries < len(a.services); retries++ {
		hash, err := a.services[a.lastUsedSvc].Verify(ctx, challenge, signature)
		if err == nil {
			return hash, nil
		}
		if errors.Is(err, types.ErrChallengeInvalid) {
			return nil, err
		}
		a.lastUsedSvc = (a.lastUsedSvc + 1) % len(a.services)
	}

	return nil, types.ErrCouldNotVerify
}

func NewRoundRobinChallengeVerifier(services []types.ChallengeVerifier) types.ChallengeVerifier {
	return &roundRobinChallengeVerifier{
		services: services,
	}
}

// cachingChallengeVerifier implements caching layer on top of
// its ChallengeVerifier.
type cachingChallengeVerifier struct {
	cache    *lru.Cache
	verifier types.ChallengeVerifier
}

type challengeVerifierResult struct {
	*types.ChallengeVerificationResult
	err error
}

func (a *cachingChallengeVerifier) Verify(ctx context.Context, challenge, signature []byte) (*types.ChallengeVerificationResult, error) {
	var challengeHash [sha256.Size]byte
	hasher := sha256.New()
	hasher.Write(challenge)
	hasher.Write(signature)
	hasher.Sum(challengeHash[:0])

	logger := logging.FromContext(ctx).WithFields(log.Field(zap.Binary("challenge", challengeHash[:])))
	if result, ok := a.cache.Get(challengeHash); ok {
		logger.Debug("retrieved challenge verifier result from the cache")
		// SAFETY: type assertion will never panic as we insert only `*ATX` values.
		result := result.(*challengeVerifierResult)
		return result.ChallengeVerificationResult, result.err
	}

	result, err := a.verifier.Verify(ctx, challenge, signature)
	if err == nil || errors.Is(err, types.ErrChallengeInvalid) {
		a.cache.Add(challengeHash, &challengeVerifierResult{ChallengeVerificationResult: result, err: err})
	}
	return result, err
}

func NewCachingChallengeVerifier(size int, verifier types.ChallengeVerifier) (types.ChallengeVerifier, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &cachingChallengeVerifier{
		cache:    cache,
		verifier: verifier,
	}, nil
}

type retryingChallengeVerifier struct {
	backoffBase       time.Duration
	backoffMultiplier float64
	maxRetries        uint
	verifier          types.ChallengeVerifier
}

func (v *retryingChallengeVerifier) Verify(ctx context.Context, challenge, signature []byte) (*types.ChallengeVerificationResult, error) {
	logger := logging.FromContext(ctx)
	timer := time.NewTimer(0)
	<-timer.C
	delay := v.backoffBase

	for retry := uint(0); retry < v.maxRetries; retry++ {
		if hash, err := v.verifier.Verify(ctx, challenge, signature); err == nil {
			return hash, nil
		} else if errors.Is(err, types.ErrChallengeInvalid) {
			return nil, err
		}
		delay = time.Duration(float64(delay) * v.backoffMultiplier)
		if delay > MaxBackoff {
			delay = MaxBackoff
		}
		timer.Reset(delay)
		logger.Info("Retrying for %d time, waiting %v", retry+1, delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, types.ErrCouldNotVerify
		}
	}
	return nil, types.ErrCouldNotVerify
}

func NewRetryingChallengeVerifier(verifier types.ChallengeVerifier, maxRetries uint, backoffBase time.Duration, backoffMultiplier float64) types.ChallengeVerifier {
	return &retryingChallengeVerifier{
		maxRetries:        maxRetries,
		verifier:          verifier,
		backoffBase:       backoffBase,
		backoffMultiplier: backoffMultiplier,
	}
}
