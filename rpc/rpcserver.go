package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/gateway/broadcaster"
	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/service"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	s          *service.Service
	gtwManager *gateway.Manager
	cfg        config.Config
	sync.Mutex
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(service *service.Service, gtwManager *gateway.Manager, cfg config.Config) *rpcServer {
	server := &rpcServer{
		s:          service,
		cfg:        cfg,
		gtwManager: gtwManager,
	}
	return server
}

func (r *rpcServer) Start(ctx context.Context, in *api.StartRequest) (*api.StartResponse, error) {
	r.Lock()
	defer r.Unlock()

	if r.s.Started() {
		return nil, service.ErrAlreadyStarted
	}

	connAcks := uint(in.ConnAcksThreshold)
	if connAcks < 1 {
		connAcks = 1
	}

	broadcastAcks := in.BroadcastAcksThreshold
	if broadcastAcks < 1 {
		broadcastAcks = 1
	}

	gtwConnCtx, cancel := context.WithTimeout(ctx, r.cfg.GtwConnTimeout)
	defer cancel()
	gtwManager, err := gateway.NewManager(gtwConnCtx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := gtwManager.Close(); err != nil {
			logging.FromContext(ctx).With().Warning("failed to close GRPC connections", log.Err(err))
		}
	}()
	b, err := broadcaster.New(
		gtwManager.Connections(),
		in.DisableBroadcast,
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	if err != nil {
		return nil, err
	}

	verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge verifier: %w", err)
	}

	// Swap the new and old gateway managers.
	// The old one will be closed in defer.
	r.gtwManager, gtwManager = gtwManager, r.gtwManager
	if err := r.s.Start(b, verifier); err != nil {
		return nil, fmt.Errorf("failed to start service: %w", err)
	}

	return &api.StartResponse{}, nil
}

func (r *rpcServer) UpdateGateway(ctx context.Context, in *api.UpdateGatewayRequest) (*api.UpdateGatewayResponse, error) {
	r.Lock()
	defer r.Unlock()

	if !r.s.Started() {
		return nil, service.ErrNotStarted
	}

	connAcks := uint(in.ConnAcksThreshold)
	if connAcks < 1 {
		connAcks = 1
	}

	broadcastAcks := in.BroadcastAcksThreshold
	if broadcastAcks < 1 {
		broadcastAcks = 1
	}

	gtwConnCtx, cancel := context.WithTimeout(ctx, r.cfg.GtwConnTimeout)
	defer cancel()
	gtwManager, err := gateway.NewManager(gtwConnCtx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := gtwManager.Close(); err != nil {
			logging.FromContext(ctx).With().Warning("failed to close GRPC connections", log.Err(err))
		}
	}()
	b, err := broadcaster.New(
		gtwManager.Connections(),
		in.DisableBroadcast,
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	if err != nil {
		return nil, err
	}

	verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge verifier: %w", err)
	}

	// Swap the new and old gateway managers.
	// The old one will be closed in defer.
	r.gtwManager, gtwManager = gtwManager, r.gtwManager
	r.s.SetBroadcaster(b)
	r.s.SetChallengeVerifier(verifier)

	return &api.UpdateGatewayResponse{}, nil
}

// Submit implements api.Submit.
func (r *rpcServer) Submit(ctx context.Context, in *api.SubmitRequest) (*api.SubmitResponse, error) {
	result, err := r.s.Submit(ctx, in.Challenge, in.Signature)
	switch {
	case errors.Is(err, service.ErrNotStarted):
		return nil, status.Error(codes.FailedPrecondition, "cannot submit a challenge because poet service is not started")
	case errors.Is(err, challenge_verifier.ErrChallengeInvalid):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, challenge_verifier.ErrCouldNotVerify):
		return nil, status.Error(codes.Unavailable, "failed to verify the challenge, consider retrying")
	case err != nil:
		logging.FromContext(ctx).With().Warning("unknown error during challenge validation", log.Err(err))
		return nil, status.Error(codes.Internal, "unknown error during challenge validation")
	}

	out := new(api.SubmitResponse)
	out.RoundId = result.Round
	out.Hash = result.Hash
	out.RoundEnd = durationpb.New(result.RoundEnd)
	return out, nil
}

// GetInfo implements api.GetInfo.
func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	info, err := r.s.Info(ctx)
	if err != nil {
		return nil, err
	}

	out := new(api.GetInfoResponse)
	out.OpenRoundId = info.OpenRoundID

	ids := make([]string, len(info.ExecutingRoundsIds))
	copy(ids, info.ExecutingRoundsIds)
	out.ExecutingRoundsIds = ids
	out.ServicePubKey = r.s.PubKey

	return out, nil
}
