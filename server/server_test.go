package server_test

// End to end tests running a Poet server and interacting with it via
// its GRPC API.

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/merkle-tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/hash"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

const randomHost = "localhost:0"

type gatewayService struct {
	pb.UnimplementedGatewayServiceServer
}

func (*gatewayService) VerifyChallenge(
	ctx context.Context,
	req *pb.VerifyChallengeRequest,
) (*pb.VerifyChallengeResponse, error) {
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
	t.Cleanup(func() { assert.NoError(t, eg.Wait()) })

	eg.Go(server.Serve)
	t.Cleanup(server.Stop)

	return server.Target()
}

func spawnPoet(ctx context.Context, t *testing.T, cfg config.Config) (*server.Server, api.PoetServiceClient) {
	t.Helper()
	req := require.New(t)

	_, err := config.SetupConfig(&cfg)
	req.NoError(err)

	srv, err := server.New(context.Background(), cfg)
	req.NoError(err)

	conn, err := grpc.DialContext(
		context.Background(),
		srv.RpcAddr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	req.NoError(err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	return srv, api.NewPoetServiceClient(conn)
}

// Test poet service startup.
func TestPoetStart(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	gtw := spawnMockGateway(t)

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
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
	cfg.Service.Genesis = time.Now().Add(time.Second).Format(time.RFC3339)
	cfg.Service.EpochDuration = time.Second
	cfg.Service.PhaseShift = 0
	cfg.Service.CycleGap = 0
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
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
	cancel()

	merkleProof := shared.MerkleProof{
		Root:         proof.Proof.Proof.Root,
		ProvenLeaves: proof.Proof.Proof.ProvenLeaves,
		ProofNodes:   proof.Proof.Proof.ProofNodes,
	}

	root, err := calcRoot(proof.Proof.Members)
	req.NoError(err)

	labelHashFunc := hash.GenLabelHashFunc(root)
	merkleHashFunc := hash.GenMerkleHashFunc(root)
	req.NoError(verifier.Validate(merkleProof, labelHashFunc, merkleHashFunc, proof.Proof.Leaves, shared.T))

	req.NoError(eg.Wait())
}

func calcRoot(leaves [][]byte) ([]byte, error) {
	tree, err := merkle.NewTree()
	if err != nil {
		return nil, fmt.Errorf("failed to generate tree: %w", err)
	}
	for _, member := range leaves {
		err := tree.AddLeaf(member)
		if err != nil {
			return nil, fmt.Errorf("failed to add leaf: %w", err)
		}
	}
	return tree.Root(), nil
}
