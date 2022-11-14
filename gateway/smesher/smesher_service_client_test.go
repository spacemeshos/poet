package smesher_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	v1 "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/gateway/smesher"
)

type postConfigMockService struct {
	v1.UnimplementedSmesherServiceServer
}

func (s *postConfigMockService) PostConfig(context.Context, *empty.Empty) (*v1.PostConfigResponse, error) {
	return &v1.PostConfigResponse{}, nil
}

func TestFetchPostConfig(t *testing.T) {
	// Arrange
	var eg errgroup.Group
	t.Cleanup(func() { require.NoError(t, eg.Wait()) })

	gtwFailing := gateway.NewMockGrpcServer(t)
	v1.RegisterSmesherServiceServer(gtwFailing.Server, &v1.UnimplementedSmesherServiceServer{})
	eg.Go(gtwFailing.Serve)
	t.Cleanup(gtwFailing.Stop)

	gtwOK := gateway.NewMockGrpcServer(t)
	v1.RegisterSmesherServiceServer(gtwOK.Server, &postConfigMockService{})
	eg.Go(gtwOK.Serve)
	t.Cleanup(gtwOK.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	gtwManager, err := gateway.NewManager(ctx, []string{gtwFailing.Target(), gtwOK.Target(), "i-dont-exist"}, 2)
	require.NoError(t, err)
	t.Cleanup(gtwManager.Close)

	// Act
	postConfig := smesher.FetchPostConfig(context.Background(), gtwManager.Connections())

	// Verify
	require.NotNil(t, postConfig)
}
