package gateway_test

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/spacemeshos/poet/gateway"
)

type mockGateway struct {
	port uint16
	*grpc.Server
}

func spawnMockGateway(t *testing.T) mockGateway {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	port, err := strconv.ParseUint(strings.TrimPrefix(lis.Addr().String(), "[::]:"), 10, 16)
	require.NoError(t, err)

	s := grpc.NewServer()

	go func() {
		err := s.Serve(lis)
		require.NoError(t, err)
	}()

	t.Cleanup(s.Stop)
	return mockGateway{
		Server: s,
		port:   uint16(port),
	}
}

func TestConnecting(t *testing.T) {
	gtw := spawnMockGateway(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	// Try to connect. one should succeed, one should fail
	conns, errors := gateway.Connect(ctx, []string{fmt.Sprintf("localhost:%d", gtw.port), "wrong-address"})
	require.Len(t, conns, 1)
	require.Len(t, errors, 1)
}
