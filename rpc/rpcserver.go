package rpc

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/service"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	registration *registration.Registration
	s            *service.Service
	phaseShift   time.Duration
	cycleGap     time.Duration
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServiceServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(
	svc *service.Service,
	registration *registration.Registration,
	phaseShift, cycleGap time.Duration,
) *rpcServer {
	return &rpcServer{
		s:            svc,
		registration: registration,
		phaseShift:   phaseShift,
		cycleGap:     cycleGap,
	}
}

func (r *rpcServer) PowParams(_ context.Context, _ *api.PowParamsRequest) (*api.PowParamsResponse, error) {
	params := r.registration.PowParams()
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

	powParams := registration.PowParams{
		Challenge:  in.GetPowParams().GetChallenge(),
		Difficulty: uint(in.GetPowParams().GetDifficulty()),
	}

	var deadline time.Time
	if in.Deadline != nil {
		deadline = in.Deadline.AsTime()
	}

	epoch, end, err := r.registration.Submit(ctx, in.Challenge, in.Pubkey, in.Nonce, powParams, deadline)
	switch {
	case errors.Is(err, registration.ErrInvalidPow) || errors.Is(err, registration.ErrInvalidPowParams):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, registration.ErrMaxMembersReached):
		return nil, status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, registration.ErrConflictingRegistration):
		return nil, status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, registration.ErrTooLateToRegister):
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, context.Canceled):
		return nil, status.Error(codes.Canceled, err.Error())
	case err != nil:
		logging.FromContext(ctx).Warn("unknown error submitting challenge", zap.Error(err))
		return nil, status.Error(codes.Internal, "unknown error submitting challenge")
	}

	out := new(api.SubmitResponse)
	out.RoundId = strconv.FormatUint(uint64(epoch), 10)
	out.RoundEnd = durationpb.New(time.Until(end))
	return out, nil
}

func (r *rpcServer) Info(ctx context.Context, in *api.InfoRequest) (*api.InfoResponse, error) {
	out := &api.InfoResponse{
		ServicePubkey: r.registration.Pubkey(),
		PhaseShift:    durationpb.New(r.phaseShift),
		CycleGap:      durationpb.New(r.cycleGap),
	}

	return out, nil
}

func (r *rpcServer) Proof(ctx context.Context, in *api.ProofRequest) (*api.ProofResponse, error) {
	proofMsg, err := r.registration.Proof(ctx, in.RoundId)
	switch {
	case errors.Is(err, registration.ErrNotFound):
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
