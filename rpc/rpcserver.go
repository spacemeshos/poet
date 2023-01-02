package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/logging"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/service"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	serviceDB  *service.ServiceDatabase
	s          *service.Service
	gtwManager *gateway.Manager
	cfg        config.Config
	sync.Mutex
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServiceServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(svc *service.Service, proofsDb *service.ServiceDatabase, gtwManager *gateway.Manager, cfg config.Config) *rpcServer {
	return &rpcServer{
		serviceDB:  proofsDb,
		s:          svc,
		cfg:        cfg,
		gtwManager: gtwManager,
	}
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

	gtwConnCtx, cancel := context.WithTimeout(ctx, r.cfg.GtwConnTimeout)
	defer cancel()
	gtwManager, err := gateway.NewManager(gtwConnCtx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := gtwManager.Close(); err != nil {
			logging.FromContext(ctx).Warn("failed to close GRPC connections", zap.Error(err))
		}
	}()

	verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge verifier: %w", err)
	}

	if err = r.s.Start(ctx, verifier); err != nil {
		return nil, err
	}
	// Swap the new and old gateway managers.
	// The old one will be closed in defer.
	r.gtwManager, gtwManager = gtwManager, r.gtwManager

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

	gtwConnCtx, cancel := context.WithTimeout(ctx, r.cfg.GtwConnTimeout)
	defer cancel()
	gtwManager, err := gateway.NewManager(gtwConnCtx, in.GatewayAddresses, connAcks)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := gtwManager.Close(); err != nil {
			logging.FromContext(ctx).Warn("failed to close GRPC connections", zap.Error(err))
		}
	}()

	verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge verifier: %w", err)
	}

	// Swap the new and old gateway managers.
	// The old one will be closed in defer.
	r.gtwManager, gtwManager = gtwManager, r.gtwManager
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
		logging.FromContext(ctx).Warn("unknown error during challenge validation", zap.Error(err))
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
	out.ServicePubkey = r.s.PubKey

	return out, nil
}

// GetProof implements api.PoetServer.
func (r *rpcServer) GetProof(ctx context.Context, in *api.GetProofRequest) (*api.GetProofResponse, error) {
	if info, err := r.s.Info(ctx); err == nil {
		if info.OpenRoundID == in.RoundId || slices.Contains(info.ExecutingRoundsIds, in.RoundId) {
			return nil, status.Error(codes.Unavailable, "round is not finished yet")
		}
	}

	proof, err := r.serviceDB.GetProof(ctx, in.RoundId)
	switch {
	case errors.Is(err, service.ErrNotFound):
		return nil, status.Error(codes.NotFound, "proof not found")
	case err == nil:
		out := api.GetProofResponse{
			Proof: &api.PoetProof{
				Proof: &api.MerkleProof{
					Root:         proof.Root,
					ProvenLeaves: proof.ProvenLeaves,
					ProofNodes:   proof.ProofNodes,
				},
				Leaves: proof.NumLeaves,
			},
			Pubkey: proof.ServicePubKey,
		}

		return &out, nil
	default:
		return nil, status.Error(codes.Internal, err.Error())
	}
}

// GetRound implements apiv1.PoetServiceServer.
func (r *rpcServer) GetRound(ctx context.Context, in *api.GetRoundRequest) (*api.GetRoundResponse, error) {
	members, err := r.serviceDB.GetRoundMembers(ctx, in.RoundId)
	switch {
	case errors.Is(err, service.ErrNotFound):
		return nil, status.Error(codes.NotFound, "round not found")
	case err == nil:
		out := api.GetRoundResponse{Members: members}

		return &out, nil
	default:
		return nil, status.Error(codes.Internal, err.Error())
	}
}
