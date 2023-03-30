package server_test

// End to end tests running a Poet server and interacting with it via
// its GRPC API.

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/spacemeshos/ed25519-recovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

const randomHost = "localhost:0"

func spawnPoet(ctx context.Context, t *testing.T, cfg config.Config) (*server.Server, api.PoetServiceClient) {
	t.Helper()
	req := require.New(t)

	_, err := config.SetupConfig(&cfg)
	req.NoError(err)

	srv, err := server.New(context.Background(), cfg)
	req.NoError(err)

	conn, err := grpc.DialContext(
		context.Background(),
		srv.GrpcAddr().String(),
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
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	resp, err := client.Info(context.Background(), &api.InfoRequest{})
	req.NoError(err)
	req.Equal("0", resp.OpenRoundId)

	cancel()
	req.NoError(eg.Wait())
}

func TestSubmitSignatureVerification(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// User credentials
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	req.NoError(err)

	// Submit challenge with invalid signature
	challenge := []byte("poet challenge")
	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: []byte{},
	})
	req.ErrorIs(err, status.Error(codes.InvalidArgument, "invalid signature"))

	signature := ed25519.Sign(privKey, challenge)
	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: signature,
	})
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
}

func TestSubmitPowVerification(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Service.InitialPowChallenge = "pow challenge"
	cfg.Service.PowDifficulty = 3

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// User credentials
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	req.NoError(err)

	// Submit challenge with valid signature but invalid pow
	challenge := []byte("poet challenge")

	signature := ed25519.Sign(privKey, challenge)
	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: signature,
	})
	req.ErrorIs(err, status.Error(codes.InvalidArgument, "invalid proof of work parameters"))

	// Submit data with valid signature and pow
	nonce, err := shared.SubmitPow(
		context.Background(),
		[]byte(cfg.Service.InitialPowChallenge),
		challenge,
		pubKey,
		cfg.Service.PowDifficulty,
	)
	req.NoError(err)

	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Nonce:     nonce,
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: signature,
		PowParams: &api.PowParams{
			Challenge:  []byte(cfg.Service.InitialPowChallenge),
			Difficulty: uint32(cfg.Service.PowDifficulty),
		},
	})
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
}

// Test submitting a challenge followed by proof generation and getting the proof via GRPC.
func TestSubmitAndGetProof(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Service.Genesis = time.Now().Add(time.Second).Format(time.RFC3339)
	cfg.Service.EpochDuration = time.Second
	cfg.Service.PhaseShift = 0
	cfg.Service.CycleGap = 0
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	srv, client := spawnPoet(ctx, t, *cfg)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// Submit a challenge
	challenge := []byte("poet challenge")
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	req.NoError(err)
	signature := ed25519.Sign(privKey, challenge)
	resp, err := client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: signature,
	})
	req.NoError(err)

	roundEnd := resp.RoundEnd.AsDuration()
	req.NotZero(roundEnd)

	// Wait for round to end
	<-time.After(roundEnd)

	// Query for the proof
	var proof *api.ProofResponse
	req.Eventually(func() bool {
		proof, err = client.Proof(context.Background(), &api.ProofRequest{RoundId: resp.RoundId})
		return err == nil
	}, time.Second, time.Millisecond*100)

	req.NotZero(proof.Proof.Leaves)
	req.Len(proof.Proof.Members, 1)
	req.Contains(proof.Proof.Members, challenge)
	cancel()

	merkleProof := shared.MerkleProof{
		Root:         proof.Proof.Proof.Root,
		ProvenLeaves: proof.Proof.Proof.ProvenLeaves,
		ProofNodes:   proof.Proof.Proof.ProofNodes,
	}

	root, err := prover.CalcTreeRoot(proof.Proof.Members)
	req.NoError(err)

	labelHashFunc := hash.GenLabelHashFunc(root)
	merkleHashFunc := hash.GenMerkleHashFunc(root)
	req.NoError(verifier.Validate(merkleProof, labelHashFunc, merkleHashFunc, proof.Proof.Leaves, shared.T))

	req.NoError(eg.Wait())
}

func TestGettingInitialPowParams(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	powChallenge := "initial challenge"
	powDifficulty := uint(77)

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Service.Genesis = time.Now().Add(time.Second).Format(time.RFC3339)
	cfg.Service.EpochDuration = time.Second
	cfg.Service.InitialPowChallenge = powChallenge
	cfg.Service.PowDifficulty = powDifficulty
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	srv, client := spawnPoet(ctx, t, *cfg)
	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	resp, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)
	req.EqualValues(powChallenge, resp.PowParams.Challenge)
	req.EqualValues(powDifficulty, resp.PowParams.Difficulty)

	cancel()
	req.NoError(eg.Wait())
}
