package rpc

import (
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/service"
	"golang.org/x/net/context"
)

// rpcServer is a gRPC, RPC front end to poet
type rpcServer struct {
	s *service.Service
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServer = (*rpcServer)(nil)

// NewRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(service *service.Service) *rpcServer {
	return &rpcServer{
		s: service,
	}
}

func (r *rpcServer) Submit(ctx context.Context, in *api.SubmitRequest) (*api.SubmitResponse, error) {
	round, err := r.s.Submit(in.Challenge)
	if err != nil {
		return nil, err
	}

	out := new(api.SubmitResponse)
	out.RoundId = round.Id
	return out, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	info := r.s.Info()

	out := new(api.GetInfoResponse)
	out.OpenRoundId = info.OpenRoundId

	ids := make([]string, len(info.ExecutingRoundsIds))
	for i, id := range info.ExecutingRoundsIds {
		ids[i] = id
	}
	out.ExecutingRoundsIds = ids
	out.ServicePubKey = r.s.PubKey

	return out, nil
}
