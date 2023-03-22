package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/shared"
)

func TestInsertAndGetProof(t *testing.T) {
	tempdir := t.TempDir()
	proofs := make(chan shared.ProofMessage)
	db, err := service.NewProofsDatabase(tempdir, proofs)
	require.NoError(t, err)

	var eg errgroup.Group
	ctx, cancel := context.WithCancel(context.Background())

	eg.Go(func() error {
		return db.Run(ctx)
	})

	proofs <- shared.ProofMessage{RoundID: "1"}
	var proof *shared.ProofMessage
	require.Eventually(t, func() bool {
		proof, err = db.Get(ctx, "1")
		return err == nil
	}, time.Second, time.Millisecond)

	require.Equal(t, "1", proof.RoundID)

	cancel()
	require.NoError(t, eg.Wait())
}
