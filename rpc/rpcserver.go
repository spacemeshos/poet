package rpc

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"

	"github.com/spacemeshos/poet/broadcaster"
	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/service"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	s *service.Service
	sync.Mutex
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ rpcapi.PoetServer = (*rpcServer)(nil)

// NewRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(service *service.Service) *rpcServer {
	return &rpcServer{
		s: service,
	}
}

func (r *rpcServer) Start(ctx context.Context, in *rpcapi.StartRequest) (*rpcapi.StartResponse, error) {
	r.Lock()
	defer r.Unlock()

	if r.s.Started() {
		return nil, service.ErrAlreadyStarted
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
	return &rpcapi.StartResponse{}, nil
}

func (r *rpcServer) UpdateGateway(ctx context.Context, in *rpcapi.UpdateGatewayRequest) (*rpcapi.UpdateGatewayResponse, error) {
	r.Lock()
	defer r.Unlock()

	if !r.s.Started() {
		return nil, service.ErrNotStarted
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

	r.s.SetBroadcaster(b)

	return &rpcapi.UpdateGatewayResponse{}, nil
}

func (r *rpcServer) Submit(ctx context.Context, in *rpcapi.SubmitRequest) (*rpcapi.SubmitResponse, error) {
	// TODO(brozansk) return an error once migration to new API is complete
	api.FromSubmitRequest(in)

	round, err := r.s.Submit(in.Challenge)
	if err != nil {
		return nil, err
	}

	out := new(rpcapi.SubmitResponse)
	out.RoundId = round.ID
	out.Hash = in.Challenge
	return out, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *rpcapi.GetInfoRequest) (*rpcapi.GetInfoResponse, error) {
	info, err := r.s.Info()
	if err != nil {
		return nil, err
	}

	out := new(rpcapi.GetInfoResponse)
	out.OpenRoundId = info.OpenRoundID

	ids := make([]string, len(info.ExecutingRoundsIds))
	copy(ids, info.ExecutingRoundsIds)
	out.ExecutingRoundsIds = ids
	out.ServicePubKey = r.s.PubKey

	return out, nil
}
