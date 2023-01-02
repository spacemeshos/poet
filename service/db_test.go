package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestServiceDb_GetRoundMembers(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	rounds := make(chan service.RoundData, 1)
	t.Cleanup(func() { close(rounds) })
	db, err := service.NewServiceDatabase(t.TempDir(), nil, rounds)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return db.Run(ctx) })

	_, err = db.GetRoundMembers(context.Background(), "")
	req.ErrorIs(err, service.ErrNotFound)

	rounds <- service.RoundData{
		Members: [][]byte{{0, 1, 2}},
	}

	var members [][]byte
	req.Eventually(func() bool {
		members, err = db.GetRoundMembers(context.Background(), "")
		return err == nil
	}, time.Second, time.Millisecond*10)
	req.Equal([][]byte{{0, 1, 2}}, members)

	_, err = db.GetProof(context.Background(), "")
	req.ErrorIs(err, service.ErrNotFound)

	cancel()
	req.NoError(eg.Wait())
}

func TestServiceDb_GetProof(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	proofs := make(chan shared.ProofMessage, 1)
	t.Cleanup(func() { close(proofs) })
	db, err := service.NewServiceDatabase(t.TempDir(), proofs, nil)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return db.Run(ctx) })

	_, err = db.GetProof(context.Background(), "")
	req.ErrorIs(err, service.ErrNotFound)

	proof := shared.ProofMessage{
		Proof: shared.Proof{
			NumLeaves: 123,
		},
		ServicePubKey: []byte("service pub key"),
	}
	proofs <- proof

	var gotProof *shared.ProofMessage
	req.Eventually(func() bool {
		gotProof, err = db.GetProof(context.Background(), "")
		return err == nil
	}, time.Second, time.Millisecond*10)
	req.Equal(proof, *gotProof)

	_, err = db.GetRoundMembers(context.Background(), "")
	req.ErrorIs(err, service.ErrNotFound)

	cancel()
	req.NoError(eg.Wait())
}
