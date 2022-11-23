package challenge_verifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/types"
	"github.com/spacemeshos/poet/types/mocks"
)

func TestRoundRobinVerifier(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	challenge := []byte("challenge")
	signature := []byte("signature")
	expected := &types.ChallengeVerificationResult{Hash: []byte("hash")}

	faultyVerifier := mocks.NewMockChallengeVerifier(ctrl)
	faultyVerifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(nil, types.ErrCouldNotVerify)
	verifier := mocks.NewMockChallengeVerifier(ctrl)
	verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(expected, nil)

	rrVerifier := challenge_verifier.NewRoundRobinChallengeVerifier([]types.ChallengeVerifier{faultyVerifier, verifier})

	result, err := rrVerifier.Verify(context.Background(), challenge, signature)
	require.NoError(t, err)
	require.EqualValues(t, expected, result)

	// Second Get should query `verifier` second time.
	result, err = rrVerifier.Verify(context.Background(), challenge, signature)
	require.NoError(t, err)
	require.EqualValues(t, expected, result)
}

func TestCachedProvider(t *testing.T) {
	t.Parallel()
	challenge := []byte("challenge")
	challenge2 := []byte("challenge2")
	signature := []byte("signature")
	expected := &types.ChallengeVerificationResult{Hash: []byte("hash")}

	t.Run("caching", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(expected, nil)

		rrProvider, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		result, err := rrProvider.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)

		result, err = rrProvider.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)
	})
	t.Run("eviction", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expected := &types.ChallengeVerificationResult{Hash: []byte("hash")}
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(expected, nil)
		verifier.EXPECT().Verify(gomock.Any(), challenge2, signature).Return(expected, nil)

		cachingVerifier, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		result, err := cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)

		result, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)

		// This Get should evict `test`
		result, err = cachingVerifier.Verify(context.Background(), challenge2, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)

		result, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)
	})
	t.Run("doesn't cache when verification failed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expected := &types.ChallengeVerificationResult{Hash: []byte("hash")}
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(nil, types.ErrCouldNotVerify)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(expected, nil)

		cachingVerifier, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		_, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.Error(t, err)
		// should not hit cache
		_, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.Error(t, err)
		result, err := cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)
		// should hit cache
		result, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)
	})
}

func TestRetryingProvider(t *testing.T) {
	t.Parallel()
	challenge := []byte("challenge")
	signature := []byte("signature")
	expected := &types.ChallengeVerificationResult{Hash: []byte("hash")}

	t.Run("eventually succeeds", func(t *testing.T) {
		t.Parallel()
		verifier := mocks.NewMockChallengeVerifier(gomock.NewController(t))
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(nil, types.ErrCouldNotVerify)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(expected, nil)
		provider := challenge_verifier.NewRetryingChallengeVerifier(verifier, 3, time.Nanosecond, 1)

		result, err := provider.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, expected, result)
	})
	t.Run("max retries reached", func(t *testing.T) {
		t.Parallel()
		verifier := mocks.NewMockChallengeVerifier(gomock.NewController(t))
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).AnyTimes().Return(nil, types.ErrCouldNotVerify)
		provider := challenge_verifier.NewRetryingChallengeVerifier(verifier, 3, time.Nanosecond, 1)

		_, err := provider.Verify(context.Background(), challenge, signature)
		require.ErrorIs(t, err, types.ErrCouldNotVerify)
	})
	t.Run("doesn't retry when the challenge is invalid", func(t *testing.T) {
		t.Parallel()
		verifier := mocks.NewMockChallengeVerifier(gomock.NewController(t))
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(nil, types.ErrChallengeInvalid)
		provider := challenge_verifier.NewRetryingChallengeVerifier(verifier, 3, time.Nanosecond, 1)

		_, err := provider.Verify(context.Background(), challenge, signature)
		require.ErrorIs(t, err, types.ErrChallengeInvalid)
	})
	t.Run("stops on ctx cancellation", func(t *testing.T) {
		t.Parallel()
		verifier := mocks.NewMockChallengeVerifier(gomock.NewController(t))
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).AnyTimes().Return(nil, types.ErrCouldNotVerify)
		provider := challenge_verifier.NewRetryingChallengeVerifier(verifier, 100000000, time.Second, 100)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.Verify(ctx, challenge, signature)
		require.ErrorIs(t, err, types.ErrCouldNotVerify)
	})
}
