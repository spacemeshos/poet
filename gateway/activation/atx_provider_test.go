package activation_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/gateway/activation"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/types"
	"github.com/spacemeshos/poet/types/mocks"
)

func TestRoundRobinProvider(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	atx := types.ATX{
		NodeID:     []byte("node-id"),
		Sequence:   7,
		PubLayerID: 99,
	}

	providerWithoutAtxId := mocks.NewMockAtxProvider(ctrl)
	providerWithoutAtxId.EXPECT().Get(gomock.Any(), shared.ATXID("test")).Return(nil, activation.ErrAtxNotFound)
	providerWithAtxId := mocks.NewMockAtxProvider(ctrl)
	providerWithAtxId.EXPECT().Get(gomock.Any(), shared.ATXID("test")).Times(2).Return(&atx, nil)

	rrProvider := activation.NewRoundRobinAtxProvider([]types.AtxProvider{providerWithoutAtxId, providerWithAtxId})

	fetchedAtx, err := rrProvider.Get(context.Background(), shared.ATXID("test"))
	require.NoError(t, err)
	require.EqualValues(t, atx, *fetchedAtx)

	// Second Get should query `providerWithAtxId` second time.
	fetchedAtx, err = rrProvider.Get(context.Background(), shared.ATXID("test"))
	require.NoError(t, err)
	require.EqualValues(t, atx, *fetchedAtx)
}

func TestCachedProvider(t *testing.T) {
	t.Parallel()
	atx := types.ATX{
		NodeID:     []byte("node-id"),
		Sequence:   7,
		PubLayerID: 99,
	}

	t.Run("caching", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockAtxProvider(ctrl)
		provider.EXPECT().Get(gomock.Any(), shared.ATXID("test")).Return(&atx, nil)

		rrProvider, err := activation.NewCachedAtxProvider(1, provider)
		require.NoError(t, err)

		fetchedAtx, err := rrProvider.Get(context.Background(), shared.ATXID("test"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)

		fetchedAtx, err = rrProvider.Get(context.Background(), shared.ATXID("test"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)
	})
	t.Run("eviction", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockAtxProvider(ctrl)
		provider.EXPECT().Get(gomock.Any(), shared.ATXID("test")).Times(2).Return(&atx, nil)
		provider.EXPECT().Get(gomock.Any(), shared.ATXID("evict")).Return(&atx, nil)

		rrProvider, err := activation.NewCachedAtxProvider(1, provider)
		require.NoError(t, err)

		fetchedAtx, err := rrProvider.Get(context.Background(), shared.ATXID("test"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)

		fetchedAtx, err = rrProvider.Get(context.Background(), shared.ATXID("test"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)

		// This Get should evict `test`
		fetchedAtx, err = rrProvider.Get(context.Background(), shared.ATXID("evict"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)

		fetchedAtx, err = rrProvider.Get(context.Background(), shared.ATXID("test"))
		require.NoError(t, err)
		require.EqualValues(t, atx, *fetchedAtx)
	})
}
