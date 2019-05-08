package rpccore

import (
	"github.com/spacemeshos/poet/hash"
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
	ErrNoProofExists = status.Error(codes.FailedPrecondition, "no computed proof exists")
)

// rpcServer is a gRPC, RPC front end to poet core
type rpcServer struct {
	s     *signal.Signal
	proof *shared.MerkleProof
}

// A compile time check to ensure that rpcServer fully implements the
// PoetCoreProverServer and PoetVerifierServer gRPC services.
var _ apicore.PoetCoreProverServer = (*rpcServer)(nil)
var _ apicore.PoetVerifierServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(s *signal.Signal) *rpcServer {
	return &rpcServer{
		s: s,
	}
}

func (r *rpcServer) Compute(ctx context.Context, in *apicore.ComputeRequest) (*apicore.ComputeResponse, error) {
	challenge := in.D.X
	leafCount := uint64(1) << in.D.N
	securityParam := shared.T
	proof, err := prover.GetProof(hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafCount, securityParam)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	r.proof = &proof

	return &apicore.ComputeResponse{Phi: proof.Root}, nil
}

func (r *rpcServer) GetNIP(ctx context.Context, in *apicore.GetNIPRequest) (*apicore.GetNIPResponse, error) {
	if r.proof == nil {
		return nil, ErrNoProofExists
	}

	return &apicore.GetNIPResponse{Proof: &apicore.Proof{
		Phi:          r.proof.Root,
		ProvenLeaves: r.proof.ProvenLeaves,
		ProofNodes:   r.proof.ProofNodes,
	}}, nil
}

func (r *rpcServer) Shutdown(context.Context, *apicore.ShutdownRequest) (*apicore.ShutdownResponse, error) {
	r.s.RequestShutdown()
	return &apicore.ShutdownResponse{}, nil
}

func nativeProofFromWire(wireProof *apicore.Proof) shared.MerkleProof {
	return shared.MerkleProof{
		Root:         wireProof.Phi,
		ProvenLeaves: wireProof.ProvenLeaves,
		ProofNodes:   wireProof.ProofNodes,
	}
}

func (r *rpcServer) VerifyNIP(ctx context.Context, in *apicore.VerifyNIPRequest) (*apicore.VerifyNIPResponse, error) {
	proof := nativeProofFromWire(in.P)
	challenge := in.D.X
	leafCount := uint64(1) << in.D.N
	securityParam := shared.T
	err := verifier.Validate(proof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafCount, securityParam)
	if err != nil {
		return nil, err
	}
	return &apicore.VerifyNIPResponse{Verified: true}, nil
}
