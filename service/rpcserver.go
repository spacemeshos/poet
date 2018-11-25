package service

import (
	"github.com/spacemeshos/poet-core-api/pcrpc"
	"github.com/spacemeshos/poet-ref/shared"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrProverExists    = status.Error(codes.FailedPrecondition, "active prover instance already exists")
	ErrNoProverExists  = status.Error(codes.FailedPrecondition, "no active prover instance exists")
	ErrInvalidHashFunc = status.Error(codes.FailedPrecondition, "invalid hash function")
)

// rpcServer is a gRPC, RPC front end to the poet core
type rpcServer struct {
	newProver   func(x []byte, n uint, h shared.HashFunc) (shared.IProver, error)
	newVerifier func(x []byte, n uint, h shared.HashFunc) (shared.IBasicVerifier, error)
	hashFuncMap map[string]func([]byte) shared.HashFunc
	prover      shared.IProver
}

// A compile time check to ensure that rpcServer fully implements the
// PoetCoreProverServer gRPC service.
var _ pcrpc.PoetCoreProverServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(
	newProver func(x []byte, n uint, h shared.HashFunc) (shared.IProver, error),
	newVerifier func(x []byte, n uint, h shared.HashFunc) (shared.IBasicVerifier, error),
	newSHA256HashFunc func(x []byte) shared.HashFunc,
	newScryptHashFunc func(x []byte) shared.HashFunc,
) *rpcServer {
	return &rpcServer{
		newProver:   newProver,
		newVerifier: newVerifier,
		hashFuncMap: map[string]func([]byte) shared.HashFunc{
			"sha256": newSHA256HashFunc,
			"scrypt": newScryptHashFunc,
		},
		prover: nil,
	}
}

func (r *rpcServer) Compute(ctx context.Context, in *pcrpc.ComputeRequest) (*pcrpc.ComputeResponse, error) {
	if r.prover != nil {
		return nil, ErrProverExists
	}

	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	var err error
	r.prover, err = r.newProver(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	phi, err := r.prover.ComputeDag()

	return &pcrpc.ComputeResponse{Phi: phi}, nil
}

func (r *rpcServer) Clean(ctx context.Context, in *pcrpc.CleanRequest) (*pcrpc.CleanResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	r.prover.DeleteStore()
	r.prover = nil

	return &pcrpc.CleanResponse{}, nil
}

func (r *rpcServer) GetNIP(ctx context.Context, in *pcrpc.GetNIPRequest) (*pcrpc.GetNIPResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	proof, err := r.prover.GetNonInteractiveProof()
	if err != nil {
		return nil, err
	}

	return &pcrpc.GetNIPResponse{Proof: &pcrpc.Proof{
		Phi: proof.Phi,
		L:   nativeLabelsToWire(proof.L),
	}}, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *pcrpc.GetProofRequest) (*pcrpc.GetProofResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	proof, err := r.prover.GetProof(shared.Challenge{Data: wireChallengeToNative(in.C)})
	if err != nil {
		return nil, err
	}

	return &pcrpc.GetProofResponse{Proof: &pcrpc.Proof{
		Phi: proof.Phi,
		L:   nativeLabelsToWire(proof.L),
	}}, nil
}

func (r *rpcServer) Shutdown(context.Context, *pcrpc.ShutdownRequest) (*pcrpc.ShutdownResponse, error) {
	RequestShutdown()
	return &pcrpc.ShutdownResponse{}, nil
}

func (r *rpcServer) VerifyProof(ctx context.Context, in *pcrpc.VerifyProofRequest) (*pcrpc.VerifyProofResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	verified := verifier.Verify(
		shared.Challenge{Data: wireChallengeToNative(in.C)},
		shared.Proof{Phi: in.P.Phi, L: wireLabelsToNative(in.P.L)},
	)
	if err != nil {
		return nil, err
	}

	return &pcrpc.VerifyProofResponse{Verified: verified}, nil
}

func (r *rpcServer) VerifyNIP(ctx context.Context, in *pcrpc.VerifyNIPRequest) (*pcrpc.VerifyNIPResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	verified, err := verifier.VerifyNIP(shared.Proof{
		Phi: in.P.Phi,
		L:   wireLabelsToNative(in.P.L),
	})
	if err != nil {
		return nil, err
	}

	return &pcrpc.VerifyNIPResponse{Verified: verified}, nil
}

func (r *rpcServer) GetRndChallenge(ctx context.Context, in *pcrpc.GetRndChallengeRequest) (*pcrpc.GetRndChallengeResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	c, err := verifier.CreteRndChallenge()
	if err != nil {
		return nil, err
	}

	return &pcrpc.GetRndChallengeResponse{C: nativeChallengeToWire(c.Data)}, nil
}

func wireLabelsToNative(in []*pcrpc.Labels) (native [shared.T]shared.Labels) {
	for i, inLabels := range in {
		var outLabels shared.Labels
		for _, inLabel := range inLabels.Labels {
			outLabels = append(outLabels, inLabel)
		}
		native[i] = outLabels
	}
	return native
}

func nativeLabelsToWire(native [shared.T]shared.Labels) (out []*pcrpc.Labels) {
	for _, labels := range native {
		var labelsMsg pcrpc.Labels
		for _, label := range labels {
			labelsMsg.Labels = append(labelsMsg.Labels, label)
		}
		out = append(out, &labelsMsg)
	}
	return out
}

func wireChallengeToNative(in []string) (native [shared.T]shared.Identifier) {
	for i, identifier := range in {
		native[i] = shared.Identifier(identifier)
	}
	return native
}

func nativeChallengeToWire(native [shared.T]shared.Identifier) (out []string) {
	for _, identifier := range native {
		out = append(out, string(identifier))
	}
	return out
}


