package rpc

import (
	"github.com/spacemeshos/poet-ref/rpc/api"
	"golang.org/x/net/context"
)

// rpcServer is a gRPC, RPC front end to poet
type rpcServer struct {}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServer = (*rpcServer)(nil)

// NewRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer() *rpcServer {
	return &rpcServer{}
}

func (r *rpcServer) GetServiceInfo(ctx context.Context, in *api.GetServiceInfoRequest) (*api.GetServiceInfoResponse, error) {
	return &api.GetServiceInfoResponse{Info: "hello world"}, nil
}
