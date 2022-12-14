package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestConnecting(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	gtw := NewMockGrpcServer(t)
	var eg errgroup.Group
	eg.Go(gtw.Serve)
	t.Cleanup(func() { require.NoError(t, eg.Wait()) })
	t.Cleanup(gtw.Stop)

	t.Run("min successful connections met", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		mgr, err := NewManager(ctx, []string{gtw.Target(), "wrong-address"}, 1)
		req.NoError(err)
		t.Cleanup(func() { require.NoError(t, mgr.Close()) })
		req.Len(mgr.Connections(), 1)
	})
	t.Run("min successful not connections met", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		mgr, err := NewManager(ctx, []string{gtw.Target(), "wrong-address"}, 2)
		req.Error(err)
		req.Nil(mgr)
	})
}
