package gateway

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockGrpcServer allows for simulating a GRPC server in tests.
// Typical usage:
// ```
// gtw := NewMockGrpcServer(t)
// <register required GRPC services>
// var eg errgroup.Group
// eg.Go(gtw.Serve)
// t.Cleanup(func() { require.NoError(t, eg.Wait()) })
// t.Cleanup(gtw.Stop)
// ```
// .
type MockGrpcServer struct {
	listener net.Listener
	*grpc.Server
}

func NewMockGrpcServer(t *testing.T) *MockGrpcServer {
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	return &MockGrpcServer{
		listener: lis,
		Server:   grpc.NewServer(),
	}
}

func (s *MockGrpcServer) Serve() error {
	return s.Server.Serve(s.listener)
}

func (s *MockGrpcServer) Target() string {
	return s.listener.Addr().String()
}
