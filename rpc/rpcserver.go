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
	"github.com/spacemeshos/poet/shared"
)

// rpcServer is a gRPC, RPC front end to poet.
type rpcServer struct {
	registration *registration.Registration
	phaseShift   time.Duration
	cycleGap     time.Duration
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServiceServer = (*rpcServer)(nil)

// NewServer creates and returns a new instance of the rpcServer.
func NewServer(
	registration *registration.Registration,
	phaseShift, cycleGap time.Duration,
) *rpcServer {
	return &rpcServer{
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

	// FIXME: PoW is deprecated
	powParams := registration.PowParams{
		Challenge:  in.GetPowParams().GetChallenge(),
		Difficulty: uint(in.GetPowParams().GetDifficulty()),
	}

	var deadline time.Time
	if in.Deadline != nil {
		deadline = in.Deadline.AsTime()
	}

	var certificate *shared.OpaqueCert
	if in.Certificate != nil {
		certificate = &shared.OpaqueCert{
			Data:      in.Certificate.GetData(),
			Signature: in.Certificate.GetSignature(),
		}
	}
	var certPubkeyHint *shared.CertKeyHint
	if in.CertificatePubkeyHint != nil {
		if len(in.CertificatePubkeyHint) != shared.CertKeyHintSize {
			return nil, status.Error(codes.InvalidArgument, "invalid certificate public key hint")
		}
		hint := shared.CertKeyHint(in.CertificatePubkeyHint)
		certPubkeyHint = &hint
	}
	epoch, end, err := r.registration.Submit(
		ctx,
		in.Challenge,
		in.Pubkey,
		in.Nonce,
		powParams,
		certPubkeyHint,
		certificate,
		deadline,
	)
	switch {
	// FIXME: remove deprecated PoW
	case errors.Is(err, registration.ErrInvalidPow) || errors.Is(err, registration.ErrInvalidPowParams):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, registration.ErrInvalidCertificate):
		return nil, status.Error(codes.Unauthenticated, err.Error())
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

func (r *rpcServer) Info(_ context.Context, _ *api.InfoRequest) (*api.InfoResponse, error) {
	var certifierResp *api.InfoResponse_Cerifier
	if certifier := r.registration.CertifierInfo(); certifier != nil {
		certifierResp = &api.InfoResponse_Cerifier{
			Url:    certifier.URL,
			Pubkey: certifier.PubKey,
		}
	}
	out := &api.InfoResponse{
		ServicePubkey: r.registration.Pubkey(),
		PhaseShift:    durationpb.New(r.phaseShift),
		CycleGap:      durationpb.New(r.cycleGap),
		Certifier:     certifierResp,
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

func (r *rpcServer) ReloadTrustedKeys(
	ctx context.Context,
	_ *api.ReloadTrustedKeysRequest,
) (*api.ReloadTrustedKeysResponse, error) {
	err := r.registration.LoadTrustedPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	return &api.ReloadTrustedKeysResponse{}, nil
}
