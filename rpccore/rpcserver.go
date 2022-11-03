package rpccore

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/release/proto/go/rpccore/apicore"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

var ErrNoProofExists = status.Error(codes.FailedPrecondition, "no computed proof exists")

// RPCServer is a gRPC, RPC front end to poet core.
type RPCServer struct {
	stop    context.CancelFunc
	datadir string
	proof   *shared.MerkleProof
}

// A compile time check to ensure that RPCServer fully implements the
// PoetCoreProverServer and PoetVerifierServer gRPC services.
var (
	_ apicore.PoetCoreProverServer = (*RPCServer)(nil)
	_ apicore.PoetVerifierServer   = (*RPCServer)(nil)
)

// NewRPCServer creates and returns a new instance of the RPCServer.
func NewRPCServer(stop context.CancelFunc, datadir string) *RPCServer {
	return &RPCServer{
		stop:    stop,
		datadir: datadir,
	}
}

func (r *RPCServer) Compute(ctx context.Context, in *apicore.ComputeRequest) (*apicore.ComputeResponse, error) {
	challenge := in.D.X
	// TODO(dshulyak) if this is actually will be used change N to Duration for clarity
	end := time.Now().Add(time.Duration(in.D.N))
	securityParam := shared.T
	_, proof, err := prover.GenerateProofWithoutPersistency(ctx, r.datadir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), end, securityParam, prover.LowestMerkleMinMemoryLayer)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	r.proof = proof

	return &apicore.ComputeResponse{}, nil
}

func (r *RPCServer) GetNIP(ctx context.Context, in *apicore.GetNIPRequest) (*apicore.GetNIPResponse, error) {
	if r.proof == nil {
		return nil, ErrNoProofExists
	}

	return &apicore.GetNIPResponse{Proof: &apicore.Proof{
		Phi:          r.proof.Root,
		ProvenLeaves: r.proof.ProvenLeaves,
		ProofNodes:   r.proof.ProofNodes,
	}}, nil
}

func (r *RPCServer) Shutdown(context.Context, *apicore.ShutdownRequest) (*apicore.ShutdownResponse, error) {
	r.stop()
	return &apicore.ShutdownResponse{}, nil
}

func nativeProofFromWire(wireProof *apicore.Proof) shared.MerkleProof {
	return shared.MerkleProof{
		Root:         wireProof.Phi,
		ProvenLeaves: wireProof.ProvenLeaves,
		ProofNodes:   wireProof.ProofNodes,
	}
}

func (r *RPCServer) VerifyNIP(ctx context.Context, in *apicore.VerifyNIPRequest) (*apicore.VerifyNIPResponse, error) {
	proof := nativeProofFromWire(in.P)
	challenge := in.D.X
	numLeaves := uint64(1) << in.D.N
	securityParam := shared.T
	err := verifier.Validate(proof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam)
	if err != nil {
		return nil, err
	}
	return &apicore.VerifyNIPResponse{Verified: true}, nil
}
