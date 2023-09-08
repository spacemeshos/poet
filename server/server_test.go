package server_test

// End to end tests running a Poet server and interacting with it via
// its GRPC API.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/registration"
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

func TestInfoEndpoint(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Round.PhaseShift = 5 * time.Minute
	cfg.Round.CycleGap = 7 * time.Minute

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	info, err := client.Info(context.Background(), &api.InfoRequest{})
	req.NoError(err)
	req.Equal("0", info.OpenRoundId)
	req.Equal("", info.GetExecutingRoundId())
	req.Equal(cfg.Round.PhaseShift, info.PhaseShift.AsDuration())
	req.Equal(cfg.Round.CycleGap, info.CycleGap.AsDuration())
	req.NotEmpty(info.ServicePubkey)

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
	cfg.Registration.PowDifficulty = 0

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

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

	powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)

	signature := ed25519.Sign(privKey, challenge)
	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		PowParams: powParams.PowParams,
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
	cfg.Registration.PowDifficulty = 3

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

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
	resp, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)
	nonce, err := shared.FindSubmitPowNonce(
		context.Background(),
		resp.PowParams.Challenge,
		challenge,
		pubKey,
		uint(resp.PowParams.Difficulty),
	)
	req.NoError(err)

	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Nonce:     nonce,
		Challenge: challenge,
		Pubkey:    pubKey,
		Signature: signature,
		PowParams: resp.PowParams,
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
	cfg.Genesis = config.Genesis(time.Now().Add(time.Second))
	cfg.Round.EpochDuration = time.Second * 2
	cfg.Round.PhaseShift = 0
	cfg.Round.CycleGap = 0
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.PowDifficulty = 0

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// Submit a challenge
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	req.NoError(err)
	challenge := []byte("poet challenge")

	powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)

	signature := ed25519.Sign(privKey, challenge)
	resp, err := client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    pubKey,
		PowParams: powParams.PowParams,
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

// Test if cannot submit more than maximum round members.
func TestCannotSubmitMoreThanMaxRoundMembers(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.MaxRoundMembers = 2
	cfg.Registration.PowDifficulty = 0

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)

	submitChallenge := func(ch []byte) error {
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		req.NoError(err)
		signature := ed25519.Sign(privKey, ch)
		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: ch,
			Pubkey:    pubKey,
			PowParams: powParams.PowParams,
			Signature: signature,
		})
		return err
	}

	// Act
	req.NoError(submitChallenge([]byte("challenge 1")))
	req.NoError(submitChallenge([]byte("challenge 2")))
	req.ErrorIs(
		submitChallenge([]byte("challenge 3")),
		status.Error(codes.ResourceExhausted, registration.ErrMaxMembersReached.Error()),
	)
	cancel()
	req.NoError(eg.Wait())
}

// Test submitting many challenges with the same nodeID.
func TestSubmittingChallengeTwice(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.PowDifficulty = 0

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})
	t.Cleanup(func() { assert.NoError(t, eg.Wait()) })

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	req.NoError(err)

	powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)

	submitChallenge := func(ch []byte) error {
		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: ch,
			Pubkey:    pubKey,
			PowParams: powParams.PowParams,
			Signature: ed25519.Sign(privKey, ch),
		})
		return err
	}

	// Act
	req.NoError(submitChallenge([]byte("challenge 1")))
	// Submitting the same challenge is OK
	req.NoError(submitChallenge([]byte("challenge 1")))
	// Submitting a different challenge with the same nodeID is not OK
	req.ErrorIs(
		submitChallenge([]byte("challenge 2")),
		status.Error(codes.AlreadyExists, registration.ErrConflictingRegistration.Error()),
	)
}

func TestPersistingPowParams(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Registration.PowDifficulty = uint(77)
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	srv, client := spawnPoet(ctx, t, *cfg)
	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	resp, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)
	req.EqualValues(cfg.Registration.PowDifficulty, resp.PowParams.Difficulty)

	powChallenge := resp.PowParams.Challenge

	cancel()
	req.NoError(eg.Wait())
	req.NoError(srv.Close())

	// Restart the server
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	srv, client = spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })
	eg.Go(func() error {
		return srv.Start(ctx)
	})
	resp, err = client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)
	req.EqualValues(cfg.Registration.PowDifficulty, resp.PowParams.Difficulty)
	req.Equal(powChallenge, resp.PowParams.Challenge)
}

func TestPersistingKeys(t *testing.T) {
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

	info, err := client.Info(context.Background(), &api.InfoRequest{})
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
	req.NoError(srv.Close())

	// Restart the server
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	srv, client = spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	eg.Go(func() error {
		return srv.Start(ctx)
	})
	info2, err := client.Info(context.Background(), &api.InfoRequest{})
	req.NoError(err)

	req.Equal(info.ServicePubkey, info2.ServicePubkey)
}

func TestLoadSubmits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, logging.New(zapcore.InfoLevel, "log.log", false))

	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Round.EpochDuration = time.Minute * 2
	cfg.Round.PhaseShift = time.Minute
	cfg.RawRPCListener = randomHost
	cfg, err := config.SetupConfig(cfg)
	req.NoError(err)

	srv, err := server.New(context.Background(), *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })
	req.NoError(err)

	concurrentSubmits := 10000
	// spawn clients
	clients := make([]api.PoetServiceClient, 0, concurrentSubmits)
	for i := 0; i < concurrentSubmits; i++ {
		conn, err := grpc.Dial(
			srv.GrpcAddr().String(),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		req.NoError(err)
		t.Cleanup(func() { assert.NoError(t, conn.Close()) })
		clients = append(clients, api.NewPoetServiceClient(conn))
	}

	// Submit challenges
	var egSrv errgroup.Group
	egSrv.Go(func() error {
		return srv.Start(ctx)
	})
	start := time.Now()
	var eg errgroup.Group

	for _, client := range clients {
		client := client
		eg.Go(func() error {
			pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
			req.NoError(err)

			challenge := make([]byte, 32)
			_, err = rand.Read(challenge)
			req.NoError(err)

			powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
			require.NoError(t, err)

			signature := ed25519.Sign(privKey, challenge)
			_, err = client.Submit(context.Background(), &api.SubmitRequest{
				Challenge: challenge,
				Pubkey:    pubKey,
				PowParams: powParams.PowParams,
				Signature: signature,
			})
			req.NoError(err)
			return nil
		})
	}
	eg.Wait()
	t.Logf("submitting %d challenges took %v", concurrentSubmits, time.Since(start))
	cancel()
	egSrv.Wait()
}
