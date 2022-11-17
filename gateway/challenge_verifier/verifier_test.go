package challenge_verifier_test

import (
	"context"
	"testing"

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

	faultyVerifier := mocks.NewMockChallengeVerifier(ctrl)
	faultyVerifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return(nil, types.ErrCouldNotVerify)
	verifier := mocks.NewMockChallengeVerifier(ctrl)
	verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return([]byte("hash"), nil)

	rrVerifier := challenge_verifier.NewRoundRobinChallengeVerifier([]types.ChallengeVerifier{faultyVerifier, verifier})

	hash, err := rrVerifier.Verify(context.Background(), challenge, signature)
	require.NoError(t, err)
	require.EqualValues(t, []byte("hash"), hash)

	// Second Get should query `verifier` second time.
	hash, err = rrVerifier.Verify(context.Background(), challenge, signature)
	require.NoError(t, err)
	require.EqualValues(t, []byte("hash"), hash)
}

func TestCachedProvider(t *testing.T) {
	t.Parallel()
	challenge := []byte("challenge")
	challenge2 := []byte("challenge2")
	signature := []byte("signature")

	t.Run("caching", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return([]byte("hash"), nil)

		rrProvider, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		hash, err := rrProvider.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)

		hash, err = rrProvider.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)
	})
	t.Run("eviction", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return([]byte("hash"), nil)
		verifier.EXPECT().Verify(gomock.Any(), challenge2, signature).Return([]byte("hash"), nil)

		cachingVerifier, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		hash, err := cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)

		hash, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)

		// This Get should evict `test`
		hash, err = cachingVerifier.Verify(context.Background(), challenge2, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)

		hash, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)
	})
	t.Run("doesn't cache when verification failed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		verifier := mocks.NewMockChallengeVerifier(ctrl)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(nil, types.ErrCouldNotVerify)
		verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Return([]byte("hash"), nil)

		cachingVerifier, err := challenge_verifier.NewCachingChallengeVerifier(1, verifier)
		require.NoError(t, err)

		_, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.Error(t, err)
		// should not hit cache
		_, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.Error(t, err)
		hash, err := cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)
		// should hit cache
		hash, err = cachingVerifier.Verify(context.Background(), challenge, signature)
		require.NoError(t, err)
		require.EqualValues(t, []byte("hash"), hash)
	})
}
