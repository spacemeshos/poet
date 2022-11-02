package smesher_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	v1 "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/stretchr/testify/require"

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
	gtwFailing := gateway.NewMockGrpcServer(t)
	v1.RegisterSmesherServiceServer(gtwFailing.Server, &v1.UnimplementedSmesherServiceServer{})
	go func() {
		require.NoError(t, gtwFailing.Serve())
	}()
	t.Cleanup(gtwFailing.Stop)

	gtwOK := gateway.NewMockGrpcServer(t)
	v1.RegisterSmesherServiceServer(gtwOK.Server, &postConfigMockService{})
	go func() {
		require.NoError(t, gtwOK.Serve())
	}()
	t.Cleanup(gtwOK.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	gtwManager, _ := gateway.NewGatewayManager(ctx, []string{gtwFailing.Target(), gtwOK.Target(), "i-dont-exist"})
	t.Cleanup(gtwManager.Close)
	require.Len(t, gtwManager.Connections(), 2)

	// Act
	postConfig := smesher.FetchPostConfig(context.Background(), gtwManager.Connections())

	// Verify
	require.NotNil(t, postConfig)
}