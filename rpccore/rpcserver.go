package rpccore

import (
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/rpccore/apicore"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/poet/verifier"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrProverExists    = status.Error(codes.FailedPrecondition, "active prover instance already exists")
	ErrNoProverExists  = status.Error(codes.FailedPrecondition, "no active prover instance exists")
	ErrInvalidHashFunc = status.Error(codes.FailedPrecondition, "invalid hash function")
)

// rpcServer is a gRPC, RPC front end to poet core
type rpcServer struct {
	s           *signal.Signal
	proof       *shared.MerkleProof
}

// A compile time check to ensure that rpcServer fully implements the
// PoetCoreProverServer and PoetVerifierServer gRPC services.
var _ apicore.PoetCoreProverServer = (*rpcServer)(nil)
var _ apicore.PoetVerifierServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(s *signal.Signal, ) *rpcServer {
	return &rpcServer{
		s:           s,
	}
}

func (r *rpcServer) Compute(ctx context.Context, in *apicore.ComputeRequest) (*apicore.ComputeResponse, error) {
	// TODO(noamnelke): use hash function based on in.D.H ?
	challenge := shared.Sha256Challenge(in.D.X)
	leafCount := uint64(1) << in.D.N
	securityParam := shared.T
	proof, err := prover.GetProof(challenge, leafCount, securityParam)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	r.proof = &proof

	return &apicore.ComputeResponse{Phi: proof.Root}, nil
}

func (r *rpcServer) Clean(ctx context.Context, in *apicore.CleanRequest) (*apicore.CleanResponse, error) {
	// TODO(noamnelke): remove this
	return &apicore.CleanResponse{}, nil
}

func (r *rpcServer) GetNIP(ctx context.Context, in *apicore.GetNIPRequest) (*apicore.GetNIPResponse, error) {
	if r.proof == nil {
		return nil, ErrNoProverExists
	}

	return &apicore.GetNIPResponse{Proof: &apicore.Proof{
		Phi:          r.proof.Root,
		ProvenLeaves: r.proof.ProvenLeaves,
		ProofNodes:   r.proof.ProofNodes,
	}}, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *apicore.GetProofRequest) (*apicore.GetProofResponse, error) {
	if r.proof == nil {
		return nil, ErrNoProverExists
	}

	return &apicore.GetProofResponse{Proof: &apicore.Proof{
		Phi:          r.proof.Root,
		ProvenLeaves: r.proof.ProvenLeaves,
		ProofNodes:   r.proof.ProofNodes,
	}}, nil
}

func (r *rpcServer) Shutdown(context.Context, *apicore.ShutdownRequest) (*apicore.ShutdownResponse, error) {
	r.s.RequestShutdown()
	return &apicore.ShutdownResponse{}, nil
}

func (r *rpcServer) VerifyProof(ctx context.Context, in *apicore.VerifyProofRequest) (*apicore.VerifyProofResponse, error) {
	// TODO(noamnelke): this is supposed to get challenge from in.C -- do we need this?
	// TODO(noamnelke): use hash function based on in.D.H ?
	proof := nativeProofFromWire(in.P)
	challenge := shared.Sha256Challenge(in.D.X)
	leafCount := uint64(1) << in.D.N
	securityParam := shared.T
	err := verifier.Validate(proof, challenge, leafCount, securityParam)
	if err != nil {
		return nil, err
	}
	return &apicore.VerifyProofResponse{Verified: true}, nil
}

func nativeProofFromWire(wireProof *apicore.Proof) shared.MerkleProof {
	return shared.MerkleProof{
		Root:         wireProof.Phi,
		ProvenLeaves: wireProof.ProvenLeaves,
		ProofNodes:   wireProof.ProofNodes,
	}
}

func (r *rpcServer) VerifyNIP(ctx context.Context, in *apicore.VerifyNIPRequest) (*apicore.VerifyNIPResponse, error) {
	// TODO(noamnelke): use hash function based on in.D.H ?
	proof := nativeProofFromWire(in.P)
	challenge := shared.Sha256Challenge(in.D.X)
	leafCount := uint64(1) << in.D.N
	securityParam := shared.T
	err := verifier.Validate(proof, challenge, leafCount, securityParam)
	if err != nil {
		return nil, err
	}
	return &apicore.VerifyNIPResponse{Verified: true}, nil
}

func (r *rpcServer) GetRndChallenge(ctx context.Context, in *apicore.GetRndChallengeRequest) (*apicore.GetRndChallengeResponse, error) {
	// TODO(noamnelke): remove this
	return nil, ErrNoProverExists
}
