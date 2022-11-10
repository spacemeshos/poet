package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestConnecting(t *testing.T) {
	gtw := NewMockGrpcServer(t)
	var eg errgroup.Group
	eg.Go(gtw.Serve)
	t.Cleanup(func() { require.NoError(t, eg.Wait()) })
	t.Cleanup(gtw.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	// Try to connect. one should succeed, one should fail
	conns, errors := connect(ctx, []string{gtw.Target(), "wrong-address"})
	require.Len(t, conns, 1)
	require.Len(t, errors, 1)
}
