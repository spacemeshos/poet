package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestInsertAndGetProof(t *testing.T) {
	tempdir := t.TempDir()
	proofs := make(chan proofMessage)
	db, err := NewProofsDatabase(tempdir, proofs)
	require.NoError(t, err)

	var eg errgroup.Group
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eg.Go(func() error {
		return db.Run(ctx)
	})

	proofs <- proofMessage{RoundID: "1"}
	var proof *proofMessage
	require.Eventually(t, func() bool {
		proof, err = db.Get(ctx, "1")
		return err == nil
	}, time.Second, time.Millisecond)

	require.Equal(t, "1", proof.RoundID)

	cancel()
	require.NoError(t, eg.Wait())
}
