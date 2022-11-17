package activation

import (
	"context"
	"fmt"

	v1 "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"google.golang.org/grpc"

	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/types"
)

type grpcAtxServiceClient struct {
	client v1.ActivationServiceClient
}

func (c *grpcAtxServiceClient) Get(ctx context.Context, id shared.ATXID) (*types.ATX, error) {
	resp, err := c.client.Get(ctx, &v1.GetRequest{
		Id: id[:],
	})
	if err != nil {
		return nil, &TransportError{
			msg:    fmt.Sprintf("failed to get ATX %x", id),
			source: err,
		}
	}
	atx := resp.GetAtx()
	if atx == nil {
		return nil, &AtxNotFoundError{
			id: id,
		}
	}
	return &types.ATX{
		NodeID:   atx.SmesherId.GetId(),
		Sequence: atx.Sequence,
	}, nil
}

func NewClient(conn *grpc.ClientConn) types.AtxProvider {
	return &grpcAtxServiceClient{
		client: v1.NewActivationServiceClient(conn),
	}
}
