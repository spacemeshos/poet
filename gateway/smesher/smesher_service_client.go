package smesher

import (
	"context"

	v1 "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/spacemeshos/poet/types"
)

type smesherServiceClient struct {
	client v1.SmesherServiceClient
}

func (c *smesherServiceClient) GetPostConfig(ctx context.Context) (*types.PostConfig, error) {
	resp, err := c.client.PostConfig(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return &types.PostConfig{
		MinNumUnits:   resp.MinNumUnits,
		MaxNumUnits:   resp.MaxNumUnits,
		BitsPerLabel:  uint8(resp.BitsPerLabel),
		LabelsPerUnit: resp.LabelsPerUnit,
		K1:            resp.K1,
		K2:            resp.K2,
	}, nil
}

func newClient(conn *grpc.ClientConn) *smesherServiceClient {
	return &smesherServiceClient{
		client: v1.NewSmesherServiceClient(conn),
	}
}

func FetchPostConfig(ctx context.Context, gateways []*grpc.ClientConn) *types.PostConfig {
	for _, gtw := range gateways {
		client := newClient(gtw)
		if cfg, err := client.GetPostConfig(ctx); err == nil {
			return cfg
		}
		if ctx.Err() != nil {
			return nil
		}
		log.With().Debug("failed to fetch post config", log.String("gateway", gtw.Target()))
	}
	return nil
}
