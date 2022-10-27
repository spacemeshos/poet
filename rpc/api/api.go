package api

import (
	"errors"

	"github.com/spacemeshos/post/shared"

	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/signing"
)

var (
	ErrOneOfNotSet      = errors.New("either initial_post or prev_atx must be set")
	ErrInvalidPublicKey = errors.New("signer.Public() should return []byte")
)

//go:generate scalegen -types InitialPost,Challenge

type InitialPost struct {
	Proof    shared.Proof
	Metadata shared.ProofMetadata
}

type Challenge struct {
	NodeID           []byte
	PositioningAtxId []byte
	PubLayerId       []byte
	// only one of InitialPost, previousATX is valid at the same time
	InitialPost   *InitialPost
	PreviousATXId []byte
}

func (d *Challenge) intoSubmitRequestData() (*rpcapi.SubmitRequest_Data, error) {
	data := &rpcapi.SubmitRequest_Data{
		NodeId:           d.NodeID,
		PositioningAtxId: d.PositioningAtxId,
		PubLayerId:       d.PubLayerId,
	}

	if d.InitialPost != nil {
		data.Prev = &rpcapi.SubmitRequest_Data_InitialPost_{
			InitialPost: &rpcapi.SubmitRequest_Data_InitialPost{
				Proof: &rpcapi.SubmitRequest_Data_InitialPost_Proof{
					Nonce:   d.InitialPost.Proof.Nonce,
					Indices: d.InitialPost.Proof.Indices,
				},
				Metadata: &rpcapi.SubmitRequest_Data_InitialPost_Metadata{
					Commitment:    d.InitialPost.Metadata.Commitment,
					Challenge:     d.InitialPost.Metadata.Challenge,
					NumUnits:      d.InitialPost.Metadata.NumUnits,
					BitsPerLabel:  uint32(d.InitialPost.Metadata.BitsPerLabel),
					LabelsPerUnit: d.InitialPost.Metadata.LabelsPerUnit,
					K1:            d.InitialPost.Metadata.K1,
					K2:            d.InitialPost.Metadata.K2,
				},
			},
		}
	} else if d.PreviousATXId != nil {
		data.Prev = &rpcapi.SubmitRequest_Data_PrevAtxId{
			PrevAtxId: d.PreviousATXId,
		}
	} else {
		return nil, ErrOneOfNotSet
	}

	return data, nil
}

func IntoSubmitRequest(d signing.Signed[Challenge]) (*rpcapi.SubmitRequest, error) {
	data, err := (*d.Data()).intoSubmitRequestData()
	if err != nil {
		return nil, err
	}
	request := &rpcapi.SubmitRequest{
		Data:      data,
		Pubkey:    d.PubKey(),
		Signature: d.Signature(),
	}

	return request, nil
}

// FromSubmitRequest constructs SignedSubmitRequestData from a probobuf message
// It verifies signature of the data.
func FromSubmitRequest(r *rpcapi.SubmitRequest) (signing.Signed[Challenge], error) {
	// Construct data
	data := Challenge{
		NodeID:           r.GetData().GetNodeId(),
		PositioningAtxId: r.GetData().GetPositioningAtxId(),
		PubLayerId:       r.GetData().GetPubLayerId(),
	}

	if initialPost := r.Data.GetInitialPost(); initialPost != nil {
		data.InitialPost = &InitialPost{
			Proof: shared.Proof{
				Nonce:   initialPost.GetProof().GetNonce(),
				Indices: initialPost.GetProof().GetIndices(),
			},

			Metadata: shared.ProofMetadata{
				Commitment:    initialPost.GetMetadata().GetCommitment(),
				Challenge:     initialPost.GetMetadata().GetChallenge(),
				NumUnits:      initialPost.GetMetadata().GetNumUnits(),
				BitsPerLabel:  uint8(initialPost.GetMetadata().GetBitsPerLabel()),
				LabelsPerUnit: initialPost.GetMetadata().GetLabelsPerUnit(),
				K1:            initialPost.GetMetadata().GetK1(),
				K2:            initialPost.GetMetadata().GetK2(),
			},
		}
	} else if prevAtx := r.Data.GetPrevAtxId(); prevAtx != nil {
		data.PreviousATXId = prevAtx
	} else {
		return nil, ErrOneOfNotSet
	}
	return signing.NewFromScaleEncodable(data, r.Signature, r.Pubkey)
}
