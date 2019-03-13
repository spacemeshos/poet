package rpc

import (
	"github.com/spacemeshos/poet-ref/rpc/api"
	"github.com/spacemeshos/poet-ref/service"
	"github.com/spacemeshos/poet-ref/shared"
	"golang.org/x/net/context"
	"time"
)

// rpcServer is a gRPC, RPC front end to poet
type rpcServer struct {
	s *service.Service
}

// A compile time check to ensure that rpcService fully implements
// the PoetServer gRPC rpc.
var _ api.PoetServer = (*rpcServer)(nil)

// NewRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(service *service.Service) *rpcServer {
	return &rpcServer{
		s: service,
	}
}

func (r *rpcServer) SubmitCommitment(ctx context.Context, in *api.SubmitCommitmentRequest) (*api.SubmitCommitmentResponse, error) {
	res, err := r.s.SubmitCommitment(in.Commitment)
	if err != nil {
		return nil, err
	}

	out := new(api.SubmitCommitmentResponse)
	out.RoundId = int32(res.RoundId)
	return out, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	info := r.s.Info()

	out := new(api.GetInfoResponse)
	out.OpenRoundId = info.OpenRoundId
	out.ExecutingRoundsIds = info.ExecutingRoundsIds
	out.ExecutedRoundsIds = info.ExecutedRoundsIds
	return out, nil
}

func (r *rpcServer) GetRoundInfo(ctx context.Context, in *api.GetRoundInfoRequest) (*api.GetRoundInfoResponse, error) {
	info, err := r.s.RoundInfo(int(in.RoundId))
	if err != nil {
		return nil, err
	}

	nip := new(api.Proof)
	nip.Phi = make([]byte, len(info.Nip.Phi))
	nip.Phi = info.Nip.Phi
	nip.L = nativeLabelsToWire(info.Nip.L)

	out := new(api.GetRoundInfoResponse)
	out.Opened = info.Opened.UnixNano() / int64(time.Millisecond)
	out.ExecuteStart = info.ExecuteStart.UnixNano() / int64(time.Millisecond)
	out.ExecuteEnd = info.ExecuteEnd.UnixNano() / int64(time.Millisecond)
	out.NumOfcommitments = int32(info.NumOfCommitments)
	out.MerkleRoot = info.MerkleRoot
	out.Nip = nip

	return out, nil
}

func (r *rpcServer) GetMembershipProof(ctx context.Context, in *api.GetMembershipProofRequest) (*api.GetMembershipProofResponse, error) {
	proof, err := r.s.MembershipProof(int(in.RoundId), in.Commitment)
	if err != nil {
		return nil, err
	}

	out := new(api.GetMembershipProofResponse)
	out.MerkleProof = proof
	return out, nil
}

func nativeLabelsToWire(native [shared.T]shared.Labels) (wire []*api.Labels) {
	for _, labels := range native {
		var labelsMsg api.Labels
		for _, label := range labels {
			labelsMsg.Labels = append(labelsMsg.Labels, label)
		}
		wire = append(wire, &labelsMsg)
	}
	return wire
}
