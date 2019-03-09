package rpc

import (
	"github.com/spacemeshos/poet-ref/rpc/api"
	"github.com/spacemeshos/poet-ref/service"
	"golang.org/x/net/context"
)

// rpcServer is a gRPC, RPC front end to poet
type rpcServer struct {
	service *service.Service
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServer = (*rpcServer)(nil)

// NewRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(service *service.Service) *rpcServer {
	return &rpcServer{
		service: service,
	}
}

func (r *rpcServer) GetServiceInfo(ctx context.Context, in *api.GetServiceInfoRequest) (*api.GetServiceInfoResponse, error) {
	return &api.GetServiceInfoResponse{Info: "hello world"}, nil
}
