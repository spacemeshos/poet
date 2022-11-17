package rpc

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/gateway/broadcaster"
	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/signing"
	"github.com/spacemeshos/poet/types"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	s          *service.Service
	gtwManager *gateway.Manager
	sync.Mutex
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ rpcapi.PoetServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(service *service.Service, gtwManager *gateway.Manager) *rpcServer {
	server := &rpcServer{
		s:          service,
		gtwManager: gtwManager,
	}
	return server
}

func (r *rpcServer) Start(ctx context.Context, in *rpcapi.StartRequest) (*rpcapi.StartResponse, error) {
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

	gtwManager, err := gateway.NewManager(ctx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer gtwManager.Close()
	b, err := broadcaster.New(
		r.gtwManager.Connections(),
		in.DisableBroadcast,
		broadcaster.DefaultBroadcastTimeout,
		uint(broadcastAcks),
	)
	if err != nil {
		return nil, err
	}

	verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
	if err != nil {
		return nil, fmt.Errorf("failed to create ATX provider: %w", err)
	}

	// Close the old connections and save the new manager.
	// The temporary manager is nil-ed to avoid closing connections in defer.
	r.gtwManager.Close()
	r.gtwManager, gtwManager = gtwManager, nil
	if err := r.s.Start(b, verifier); err != nil {
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

	connAcks := uint(in.ConnAcksThreshold)
	if connAcks < 1 {
		connAcks = 1
	}

	broadcastAcks := in.BroadcastAcksThreshold
	if broadcastAcks < 1 {
		broadcastAcks = 1
	}

	gtwManager, err := gateway.NewManager(ctx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer gtwManager.Close()
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
		return nil, fmt.Errorf("failed to create ATX provider: %w", err)
	}

	// Close the old connections and save the new manager.
	// The temporary manager is nil-ed to avoid closing connections in defer.
	r.gtwManager.Close()
	r.gtwManager, gtwManager = gtwManager, nil
	r.s.SetBroadcaster(b)
	r.s.SetChallengeVerifier(verifier)

	return &rpcapi.UpdateGatewayResponse{}, nil
}

func (r *rpcServer) Submit(ctx context.Context, in *rpcapi.SubmitRequest) (*rpcapi.SubmitResponse, error) {
	// Temporarily support both the old and new challenge submission API.
	// TODO(brozansk) remove support for data []byte after go-spacemesh is updated to
	// use the new API.
	var challenge signing.Signed[[]byte]
	if in.Signature != nil {
		signed, err := signing.NewFromBytes(in.Challenge, in.Signature)
		if err != nil {
			return nil, err
		}
		challenge = signed
	}

	round, hash, err := r.s.Submit(ctx, in.Challenge, challenge)
	if err != nil {
		if errors.Is(err, service.ErrNotStarted) {
			return nil, status.Error(codes.FailedPrecondition, "cannot submit a challenge because poet service is not started")
		}
		if errors.Is(err, types.ErrChallengeInvalid) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		} else if errors.Is(err, types.ErrCouldNotVerify) {
			return nil, status.Error(codes.Unavailable, "failed to verify the challenge, consider retrying")
		}
		return nil, status.Error(codes.Internal, "unknown error during challenge validation")
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
