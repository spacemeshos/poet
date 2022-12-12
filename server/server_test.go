package server_test

// End to end tests running a Poet server and interacting with it via
// its GRPC API.

import (
	"context"
	"testing"
	"time"

	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/server"
)

type gatewayService struct {
	pb.UnimplementedGatewayServiceServer
}

func (*gatewayService) VerifyChallenge(ctx context.Context, req *pb.VerifyChallengeRequest) (*pb.VerifyChallengeResponse, error) {
	return &pb.VerifyChallengeResponse{
		Hash:   []byte("hash"),
		NodeId: []byte("nodeID"),
	}, nil
}

func spawnMockGateway(t *testing.T) (target string) {
	t.Helper()
	server := gateway.NewMockGrpcServer(t)
	pb.RegisterGatewayServiceServer(server.Server, &gatewayService{})

	var eg errgroup.Group
	t.Cleanup(func() { require.NoError(t, eg.Wait()) })

	eg.Go(server.Serve)
	t.Cleanup(server.Stop)

	return server.Target()
}

func spawnPoet(ctx context.Context, t *testing.T, cfg config.Config) (*server.Server, api.PoetClient) {
	t.Helper()
	req := require.New(t)

	_, err := config.SetupConfig(&cfg)
	req.NoError(err)

	srv, err := server.New(cfg)
	req.NoError(err)

	conn, err := grpc.DialContext(
		context.Background(),
		srv.RpcAddr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	req.NoError(err)
	t.Cleanup(func() { conn.Close() })

	return srv, api.NewPoetClient(conn)
}

// Test poet service startup.
func TestPoetStart(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	gtw := spawnMockGateway(t)

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = "localhost:0"
	cfg.RawRESTListener = "localhost:0"
	cfg.Service.GatewayAddresses = []string{gtw}

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	resp, err := client.GetInfo(context.Background(), &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal("0", resp.OpenRoundId)

	cancel()
	req.NoError(eg.Wait())
}

// Test submitting a challenge followed by proof generation and getting the proof via GRPC.
func TestSubmitAndGetProof(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	gtw := spawnMockGateway(t)

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Service.Genesis = time.Now().Format(time.RFC3339)
	cfg.Service.EpochDuration = time.Second
	cfg.Service.PhaseShift = 0
	cfg.Service.CycleGap = 0
	cfg.RawRPCListener = "localhost:0"
	cfg.RawRESTListener = "localhost:0"
	cfg.Service.GatewayAddresses = []string{gtw}

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// Submit a challenge
	resp, err := client.Submit(context.Background(), &api.SubmitRequest{})
	req.NoError(err)
	req.Equal([]byte("hash"), resp.Hash)

	roundEnd := resp.RoundEnd.AsDuration()
	req.NotZero(roundEnd)

	// Wait for round to end
	<-time.After(roundEnd)

	// Query for the proof
	var proof *api.GetProofResponse
	req.Eventually(func() bool {
		proof, err = client.GetProof(context.Background(), &api.GetProofRequest{RoundId: resp.RoundId})
		return err == nil
	}, time.Second, time.Millisecond*100)

	req.NotZero(proof.Proof.Leaves)
	req.Len(proof.Proof.Members, 1)
	req.Contains(proof.Proof.Members, []byte("hash"))
	req.NotEmpty(proof.Pubkey)
	cancel()
	req.NoError(eg.Wait())
}
