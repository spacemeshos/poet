package rpc

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/poet/broadcaster"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/service"
	"golang.org/x/net/context"
	"sync"
)

// rpcServer is a gRPC, RPC front end to poet
type rpcServer struct {
	s *service.Service
	sync.Mutex
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

func (r *rpcServer) Start(ctx context.Context, in *api.StartRequest) (*api.StartResponse, error) {
	r.Lock()
	defer r.Unlock()

	if r.s.Started() {
		return nil, errors.New("already started")
	}

	connAcks := in.ConnAcksThreshold
	if connAcks < 1 {
		connAcks = 1
	}

	broadcastAcks := in.BroadcastAcksThreshold
	if broadcastAcks < 1 {
		broadcastAcks = 1
	}

	b, err := broadcaster.New(
		in.GatewayAddresses,
		in.DisableBroadcast,
		broadcaster.DefaultConnTimeout,
		uint(connAcks),
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	if err != nil {
		return nil, err
	}

	if err := r.s.Start(b); err != nil {
		return nil, fmt.Errorf("failed to start service: %v", err)
	}
	return &api.StartResponse{}, nil
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
	info, err := r.s.Info()
	if err != nil {
		return nil, err
	}

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
