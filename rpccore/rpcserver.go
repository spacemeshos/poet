package rpccore

import (
	"fmt"
	"github.com/spacemeshos/poet-ref/rpccore/apicore"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/spacemeshos/poet-ref/signal"
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
	newProver   func(x []byte, n uint, h shared.HashFunc) (shared.IProver, error)
	newVerifier func(x []byte, n uint, h shared.HashFunc) (shared.IBasicVerifier, error)
	hashFuncMap map[string]func([]byte) shared.HashFunc
	prover      shared.IProver
}

// A compile time check to ensure that rpcServer fully implements the
// PoetCoreProverServer and PoetVerifierServer gRPC services.
var _ apicore.PoetCoreProverServer = (*rpcServer)(nil)
var _ apicore.PoetVerifierServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(
	s *signal.Signal,
	newProver func(x []byte, n uint, h shared.HashFunc) (shared.IProver, error),
	newVerifier func(x []byte, n uint, h shared.HashFunc) (shared.IBasicVerifier, error),
	newSHA256HashFunc func(x []byte) shared.HashFunc,
	newScryptHashFunc func(x []byte) shared.HashFunc,
) *rpcServer {
	return &rpcServer{
		s:           s,
		newProver:   newProver,
		newVerifier: newVerifier,
		hashFuncMap: map[string]func([]byte) shared.HashFunc{
			"sha256": newSHA256HashFunc,
			"scrypt": newScryptHashFunc,
		},
		prover: nil,
	}
}

func (r *rpcServer) Compute(ctx context.Context, in *apicore.ComputeRequest) (*apicore.ComputeResponse, error) {
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

	return &apicore.ComputeResponse{Phi: phi}, nil
}

func (r *rpcServer) Clean(ctx context.Context, in *apicore.CleanRequest) (*apicore.CleanResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	r.prover.DeleteStore()
	r.prover = nil

	return &apicore.CleanResponse{}, nil
}

func (r *rpcServer) GetNIP(ctx context.Context, in *apicore.GetNIPRequest) (*apicore.GetNIPResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	proof, err := r.prover.GetNonInteractiveProof()
	if err != nil {
		return nil, err
	}

	return &apicore.GetNIPResponse{Proof: &apicore.Proof{
		Phi: proof.Phi,
		L:   nativeLabelsToWire(proof.L),
	}}, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *apicore.GetProofRequest) (*apicore.GetProofResponse, error) {
	if r.prover == nil {
		return nil, ErrNoProverExists
	}

	proof, err := r.prover.GetProof(shared.Challenge{Data: wireChallengeToNative(in.C)})
	if err != nil {
		return nil, err
	}

	return &apicore.GetProofResponse{Proof: &apicore.Proof{
		Phi: proof.Phi,
		L:   nativeLabelsToWire(proof.L),
	}}, nil
}

func (r *rpcServer) Shutdown(context.Context, *apicore.ShutdownRequest) (*apicore.ShutdownResponse, error) {
	r.s.RequestShutdown()
	return &apicore.ShutdownResponse{}, nil
}

func (r *rpcServer) VerifyProof(ctx context.Context, in *apicore.VerifyProofRequest) (*apicore.VerifyProofResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	labels, err := wireLabelsToNative(in.P.L)
	if err != nil {
		return nil, err
	}

	verified := verifier.Verify(
		shared.Challenge{Data: wireChallengeToNative(in.C)},
		shared.Proof{Phi: in.P.Phi, L: *labels},
	)
	if err != nil {
		return nil, err
	}

	return &apicore.VerifyProofResponse{Verified: verified}, nil
}

func (r *rpcServer) VerifyNIP(ctx context.Context, in *apicore.VerifyNIPRequest) (*apicore.VerifyNIPResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	labels, err := wireLabelsToNative(in.P.L)
	if err != nil {
		return nil, err
	}

	verified, err := verifier.VerifyNIP(shared.Proof{
		Phi: in.P.Phi,
		L:   *labels,
	})
	if err != nil {
		return nil, err
	}

	return &apicore.VerifyNIPResponse{Verified: verified}, nil
}

func (r *rpcServer) GetRndChallenge(ctx context.Context, in *apicore.GetRndChallengeRequest) (*apicore.GetRndChallengeResponse, error) {
	hashFunc, ok := r.hashFuncMap[in.D.H]
	if !ok {
		return nil, ErrInvalidHashFunc
	}

	verifier, err := r.newVerifier(in.D.X, uint(in.D.N), hashFunc(in.D.X))
	if err != nil {
		return nil, err
	}

	c, err := verifier.CreateRndChallenge()
	if err != nil {
		return nil, err
	}

	return &apicore.GetRndChallengeResponse{C: nativeChallengeToWire(c.Data)}, nil
}

func wireLabelsToNative(wire []*apicore.Labels) (native *[shared.T]shared.Labels, err error) {
	if len(wire) != shared.T {
		return nil, fmt.Errorf("invalid number of labels, expected: %v, found: %v", shared.T, len(wire))
	}

	native = new([shared.T]shared.Labels)
	for i, inLabels := range wire {
		var outLabels shared.Labels
		for _, inLabel := range inLabels.Labels {
			outLabels = append(outLabels, inLabel)
		}
		native[i] = outLabels
	}

	return native, nil
}

func nativeLabelsToWire(native [shared.T]shared.Labels) (wire []*apicore.Labels) {
	for _, labels := range native {
		var labelsMsg apicore.Labels
		for _, label := range labels {
			labelsMsg.Labels = append(labelsMsg.Labels, label)
		}
		wire = append(wire, &labelsMsg)
	}
	return wire
}

func wireChallengeToNative(wire []string) (native [shared.T]shared.Identifier) {
	for i, identifier := range wire {
		native[i] = shared.Identifier(identifier)
	}
	return
}

func nativeChallengeToWire(native [shared.T]shared.Identifier) (wire []string) {
	for _, identifier := range native {
		wire = append(wire, string(identifier))
	}
	return wire
}
