package rpc

import (
	"fmt"
	"sync"

	"github.com/spacemeshos/smutil/log"
	"golang.org/x/net/context"

	"github.com/spacemeshos/poet/broadcaster"
	"github.com/spacemeshos/poet/gateway"
	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signing"
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

	connCtx, cancel := context.WithTimeout(ctx, broadcaster.DefaultConnTimeout)
	b, err := broadcaster.New(
		connCtx,
		in.GatewayAddresses,
		in.DisableBroadcast,
		uint(connAcks),
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	cancel()
	if err != nil {
		return nil, err
	}

	gateways, errs := gateway.Connect(ctx, in.GatewayAddresses)
	for _, err := range errs {
		log.With().Warning("failed to connect to gateway grpc server", log.Err(err))
	}
	atxProvider, err := service.CreateAtxProvider(gateways)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATX provider: %w", err)
	}

	if err := r.s.Start(b, atxProvider); err != nil {
		return nil, fmt.Errorf("failed to start service: %w", err)
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

	connCtx, cancel := context.WithTimeout(ctx, broadcaster.DefaultConnTimeout)
	b, err := broadcaster.New(
		connCtx,
		in.GatewayAddresses,
		in.DisableBroadcast,
		uint(connAcks),
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	cancel()
	if err != nil {
		return nil, err
	}

	gateways, errs := gateway.Connect(ctx, in.GatewayAddresses)
	for _, err := range errs {
		log.With().Warning("failed to connect to gateway grpc server", log.Err(err))
	}
	atxProvider, err := service.CreateAtxProvider(gateways)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATX provider: %w", err)
	}

	r.s.SetBroadcaster(b)
	r.s.SetAtxProvider(atxProvider)

	return &rpcapi.UpdateGatewayResponse{}, nil
}

func (r *rpcServer) Submit(ctx context.Context, in *rpcapi.SubmitRequest) (*rpcapi.SubmitResponse, error) {
	// Temporarily support both the old and new challenge submission API.
	// TODO(brozansk) remove support for data []byte after go-spacemesh is updated to
	// use the new API.
	var challenge signing.Signed[shared.Challenge]
	if in.Data != nil {
		signed, err := api.FromSubmitRequest(in)
		if err != nil {
			return nil, err
		}
		challenge = signed
	}

	round, hash, err := r.s.Submit(ctx, in.Challenge, challenge)
	if err != nil {
		return nil, err
	}

	out := new(rpcapi.SubmitResponse)
	out.RoundId = round.ID
	out.Hash = hash
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
