package rpc

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"sync"

	"go.uber.org/zap"
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

func (r *rpcServer) PowParams(_ context.Context, _ *api.PowParamsRequest) (*api.PowParamsResponse, error) {
	params := r.s.PowParams()
	return &api.PowParamsResponse{
		PowParams: &api.PowParams{
			Challenge:  params.Challenge,
			Difficulty: uint32(params.Difficulty),
		},
	}, nil
}

func (r *rpcServer) Submit(ctx context.Context, in *api.SubmitRequest) (*api.SubmitResponse, error) {
	if len(in.Pubkey) != ed25519.PublicKeySize {
		return nil, status.Error(codes.InvalidArgument, "invalid public key")
	}

	if !ed25519.Verify(in.Pubkey, bytes.Join([][]byte{in.Prefix, in.Challenge}, nil), in.Signature) {
		return nil, status.Error(codes.InvalidArgument, "invalid signature")
	}

	powParams := service.PowParams{
		Challenge:  in.GetPowParams().GetChallenge(),
		Difficulty: uint(in.GetPowParams().GetDifficulty()),
	}

	result, err := r.s.Submit(ctx, in.Challenge, in.Pubkey, in.Nonce, powParams)
	switch {
	case errors.Is(err, service.ErrNotStarted):
		return nil, status.Error(
			codes.FailedPrecondition,
			"cannot submit a challenge because poet service is not started",
		)
	case errors.Is(err, service.ErrInvalidPow) || errors.Is(err, service.ErrInvalidPowParams):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrMaxMembersReached):
		return nil, status.Error(codes.ResourceExhausted, err.Error())
	case err != nil:
		logging.FromContext(ctx).Warn("unknown error submitting challenge", zap.Error(err))
		return nil, status.Error(codes.Internal, "unknown error submitting challenge")
	}

	out := new(api.SubmitResponse)
	out.RoundId = result.Round
	out.RoundEnd = durationpb.New(result.RoundEnd)
	return out, nil
}

func (r *rpcServer) Info(ctx context.Context, in *api.InfoRequest) (*api.InfoResponse, error) {
	info, err := r.s.Info(ctx)
	if err != nil {
		return nil, err
	}

	out := &api.InfoResponse{
		OpenRoundId:   info.OpenRoundID,
		ServicePubkey: r.s.PubKey,
		PhaseShift:    durationpb.New(r.cfg.Service.PhaseShift),
		CycleGap:      durationpb.New(r.cfg.Service.CycleGap),
	}

	if info.ExecutingRoundId != nil {
		out.ExecutingRoundsIds = []string{*info.ExecutingRoundId}
	}

	return out, nil
}

func (r *rpcServer) Proof(ctx context.Context, in *api.ProofRequest) (*api.ProofResponse, error) {
	proofMsg, err := r.proofsDb.Get(ctx, in.RoundId)
	switch {
	case errors.Is(err, service.ErrNotFound):
		return nil, status.Error(codes.NotFound, "proof not found")
	case err == nil:
		proof := proofMsg.Proof
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
