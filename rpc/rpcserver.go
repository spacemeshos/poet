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

func (r *rpcServer) Submit(ctx context.Context, in *api.SubmitRequest) (*api.SubmitResponse, error) {
	res, err := r.s.Submit(in.Challenge)
	if err != nil {
		return nil, err
	}

	out := new(api.SubmitResponse)
	out.RoundId = int32(res.Id)
	return out, nil
}

func (r *rpcServer) GetMembershipProof(ctx context.Context, in *api.GetMembershipProofRequest) (*api.GetMembershipProofResponse, error) {
	mproof, err := r.s.MembershipProof(int(in.RoundId), in.Commitment, in.Wait)
	if err != nil {
		return nil, err
	}

	out := new(api.GetMembershipProofResponse)
	out.Mproof = new(api.MembershipProof)
	out.Mproof.Index = int32(mproof.Index)
	out.Mproof.Root = mproof.Root
	out.Mproof.Proof = mproof.Proof
	return out, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *api.GetProofRequest) (*api.GetProofResponse, error) {
	p, err := r.s.Proof(int(in.RoundId), in.Wait)
	if err != nil {
		return nil, err
	}

	pOut := new(api.Proof)
	pOut.Phi = make([]byte, len(p.Phi))
	pOut.Phi = p.Phi
	pOut.L = nativeLabelsToWire(p.L)

	out := new(api.GetProofResponse)
	out.Proof = pOut
	return out, nil
}

func (r *rpcServer) GetRoundInfo(ctx context.Context, in *api.GetRoundInfoRequest) (*api.GetRoundInfoResponse, error) {
	info, err := r.s.RoundInfo(int(in.RoundId))
	if err != nil {
		return nil, err
	}

	pOut := new(api.Proof)
	pOut.Phi = make([]byte, len(info.Nip.Phi))
	pOut.Phi = info.Nip.Phi
	pOut.L = nativeLabelsToWire(info.Nip.L)

	out := new(api.GetRoundInfoResponse)
	out.Opened = info.Opened.UnixNano() / int64(time.Millisecond)
	out.ExecuteStart = info.ExecuteStart.UnixNano() / int64(time.Millisecond)
	out.ExecuteEnd = info.ExecuteEnd.UnixNano() / int64(time.Millisecond)
	out.NumOfcommitments = int32(info.NumOfCommits)
	out.MerkleRoot = info.MerkleRoot
	out.Proof = pOut

	return out, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	info := r.s.Info()

	out := new(api.GetInfoResponse)
	out.OpenRoundId = int32(info.OpenRoundId)

	ids := make([]int32, len(info.ExecutingRoundsIds))
	for i, id := range info.ExecutingRoundsIds {
		ids[i] = int32(id)
	}
	out.ExecutingRoundsIds = ids

	ids = make([]int32, len(info.ExecutedRoundsIds))
	for i, id := range info.ExecutedRoundsIds {
		ids[i] = int32(id)
	}
	out.ExecutedRoundsIds = ids

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
