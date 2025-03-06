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
		rounds := inMemory.RegisterForRoundClosed(t.Context())
		require.NoError(t, inMemory.ExecuteRound(t.Context(), 1, []byte{1, 2, 3}))
		round := <-rounds
		require.Equal(t, uint(1), round.Epoch)
		require.Equal(t, []byte{1, 2, 3}, round.MembershipRoot)
	})
	t.Run("execute round (cancel on context canceled)", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		require.NoError(t, inMemory.ExecuteRound(t.Context(), 1, []byte{1, 2, 3}))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, inMemory.ExecuteRound(ctx, 1, []byte{1, 2, 3}), context.Canceled)
	})
	t.Run("new proof", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		proofs := inMemory.RegisterForProofs(t.Context())
		require.NoError(t, inMemory.NewProof(t.Context(), shared.NIP{
			Epoch:  1,
			Leaves: 2,
		}))
		proof := <-proofs
		require.Equal(t, uint(1), proof.Epoch)
		require.Equal(t, uint64(2), proof.Leaves)
	})
	t.Run("execute round (cancel on context canceled)", func(t *testing.T) {
		inMemory := transport.NewInMemory()
		require.NoError(t, inMemory.NewProof(t.Context(), shared.NIP{}))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, inMemory.NewProof(ctx, shared.NIP{}), context.Canceled)
	})
}
