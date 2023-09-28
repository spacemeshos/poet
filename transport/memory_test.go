package transport_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/transport"
)

func TestInMemoryTransport(t *testing.T) {
	t.Run("execute round", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		rounds := inMemory.RegisterForRoundClosed(context.Background())
		require.NoError(t, inMemory.ExecuteRound(context.Background(), 1, []byte{1, 2, 3}))
		round := <-rounds
		require.Equal(t, uint(1), round.Epoch)
		require.Equal(t, []byte{1, 2, 3}, round.MembershipRoot)
	})
	t.Run("execute round (cancel on context canceled)", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		require.NoError(t, inMemory.ExecuteRound(context.Background(), 1, []byte{1, 2, 3}))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		require.ErrorIs(t, inMemory.ExecuteRound(ctx, 1, []byte{1, 2, 3}), context.Canceled)
	})
	t.Run("new proof", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		proofs := inMemory.RegisterForProofs(context.Background())
		require.NoError(t, inMemory.NewProof(context.Background(), shared.NIP{
			Epoch:  1,
			Leaves: 2,
		}))
		proof := <-proofs
		require.Equal(t, uint(1), proof.Epoch)
		require.Equal(t, uint64(2), proof.Leaves)
	})
	t.Run("execute round (cancel on context canceled)", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		require.NoError(t, inMemory.NewProof(context.Background(), shared.NIP{}))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		require.ErrorIs(t, inMemory.NewProof(ctx, shared.NIP{}), context.Canceled)
	})
}
