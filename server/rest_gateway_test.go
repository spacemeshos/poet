package server_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/server"
)

func TestConfiguringMaxGrpcRespSize(t *testing.T) {
	req := require.New(t)
	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.MaxGrpcRespSize = 16
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	ctx := logging.NewContext(context.Background(), zaptest.NewLogger(t))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := spawnPoetServer(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})
	t.Cleanup(func() { assert.NoError(t, eg.Wait()) })

	resp, err := http.Get("http://" + srv.GrpcRestProxyAddr().String() + "/v1/info")
	req.NoError(err)

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	req.NoError(err)
	req.Contains(string(data), "received message larger than max")
}
