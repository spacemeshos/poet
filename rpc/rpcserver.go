package rpc

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/logging"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/service"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	proofsDb *service.ProofsDatabase
	s        *service.Service
	cfg      config.Config
	sync.Mutex
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServiceServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(
	svc *service.Service,
	proofsDb *service.ProofsDatabase,
	cfg config.Config,
) *rpcServer {
	return &rpcServer{
		proofsDb: proofsDb,
		s:        svc,
		cfg:      cfg,
	}
}

// Submit implements api.Submit.
func (r *rpcServer) Submit(ctx context.Context, in *api.SubmitRequest) (*api.SubmitResponse, error) {
	// TODO(poszu): verify signature
	result, err := r.s.Submit(ctx, in.Challenge, in.Pubkey, in.Nonce)
	switch {
	case errors.Is(err, service.ErrNotStarted):
		return nil, status.Error(
			codes.FailedPrecondition,
			"cannot submit a challenge because poet service is not started",
		)
	case errors.Is(err, service.ErrChallengeInvalid):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case err != nil:
		logging.FromContext(ctx).Warn("unknown error submitting challenge", zap.Error(err))
		return nil, status.Error(codes.Internal, "unknown error submitting challenge")
	}

	out := new(api.SubmitResponse)
	out.RoundId = result.Round
	out.RoundEnd = durationpb.New(result.RoundEnd)
	return out, nil
}

// GetInfo implements api.GetInfo.
func (r *rpcServer) Info(ctx context.Context, in *api.InfoRequest) (*api.InfoResponse, error) {
	info, err := r.s.Info(ctx)
	if err != nil {
		return nil, err
	}

	out := new(api.InfoResponse)
	out.OpenRoundId = info.OpenRoundID

	ids := make([]string, len(info.ExecutingRoundsIds))
	copy(ids, info.ExecutingRoundsIds)
	out.ExecutingRoundsIds = ids
	out.ServicePubkey = r.s.PubKey

	return out, nil
}

// GetProof implements api.PoetServer.
func (r *rpcServer) Proof(ctx context.Context, in *api.ProofRequest) (*api.ProofResponse, error) {
	if info, err := r.s.Info(ctx); err == nil {
		if info.OpenRoundID == in.RoundId || slices.Contains(info.ExecutingRoundsIds, in.RoundId) {
			return nil, status.Error(codes.Unavailable, "round is not finished yet")
		}
	}

	proofMsg, err := r.proofsDb.Get(ctx, in.RoundId)
	proof := proofMsg.Proof
	switch {
	case errors.Is(err, service.ErrNotFound):
		return nil, status.Error(codes.NotFound, "proof not found")
	case err == nil:
		out := api.ProofResponse{
			Proof: &api.PoetProof{
				Proof: &api.MerkleProof{
					Root:         proof.Root,
					ProvenLeaves: proof.ProvenLeaves,
					ProofNodes:   proof.ProofNodes,
				},
				Members: proof.Members,
				Leaves:  proof.NumLeaves,
			},
			Pubkey: proofMsg.ServicePubKey,
		}

		return &out, nil
	default:
		return nil, status.Error(codes.Internal, err.Error())
	}
}
