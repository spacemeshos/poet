package server_test

// End to end tests running a Poet server and interacting with it via
// its GRPC API.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

const randomHost = "localhost:0"

func spawnPoetServer(ctx context.Context, t *testing.T, cfg server.Config) *server.Server {
	t.Helper()

	server.SetupConfig(&cfg)
	srv, err := server.New(ctx, cfg)
	require.NoError(t, err)

	return srv
}

func spawnPoet(ctx context.Context, t *testing.T, cfg server.Config) (*server.Server, api.PoetServiceClient) {
	t.Helper()

	srv := spawnPoetServer(ctx, t, cfg)
	conn, err := grpc.NewClient(
		srv.GrpcAddr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	return srv, api.NewPoetServiceClient(conn)
}

func TestInfoEndpoint(t *testing.T) {
	t.Parallel()

	cfg := server.DefaultConfig()
	cfg.DisableWorker = true

	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Round.PhaseShift = 5 * time.Minute
	cfg.Round.CycleGap = 7 * time.Minute

	t.Run("no certifier", func(t *testing.T) {
		req := require.New(t)
		cfg := cfg
		cfg.PoetDir = t.TempDir()

		ctx, cancel := context.WithCancel(logging.NewContext(context.Background(), zaptest.NewLogger(t)))
		defer cancel()
		srv, client := spawnPoet(ctx, t, *cfg)
		t.Cleanup(func() { assert.NoError(t, srv.Close()) })

		var eg errgroup.Group
		eg.Go(func() error {
			return srv.Start(ctx)
		})

		info, err := client.Info(context.Background(), &api.InfoRequest{})
		req.NoError(err)
		req.Equal(cfg.Round.PhaseShift, info.PhaseShift.AsDuration())
		req.Equal(cfg.Round.CycleGap, info.CycleGap.AsDuration())
		req.NotEmpty(info.ServicePubkey)
		req.Nil(info.Certifier)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("with certifier", func(t *testing.T) {
		req := require.New(t)
		cfg := cfg
		cfg.PoetDir = t.TempDir()
		cfg.Registration.Certifier = &registration.CertifierConfig{
			URL:    "http://localhost:8080",
			PubKey: []byte("certifier pubkey"),
		}

		ctx, cancel := context.WithCancel(logging.NewContext(context.Background(), zaptest.NewLogger(t)))
		defer cancel()
		srv, client := spawnPoet(ctx, t, *cfg)
		t.Cleanup(func() { assert.NoError(t, srv.Close()) })

		var eg errgroup.Group
		eg.Go(func() error {
			return srv.Start(ctx)
		})

		info, err := client.Info(context.Background(), &api.InfoRequest{})
		req.NoError(err)
		req.Equal(cfg.Round.PhaseShift, info.PhaseShift.AsDuration())
		req.Equal(cfg.Round.CycleGap, info.CycleGap.AsDuration())
		req.NotEmpty(info.ServicePubkey)
		req.Equal(info.Certifier.Pubkey, cfg.Registration.Certifier.PubKey.Bytes())
		req.Equal(info.Certifier.Url, cfg.Registration.Certifier.URL)

		cancel()
		req.NoError(eg.Wait())
	})
}

func TestSubmitSignatureVerification(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.DisableWorker = true
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

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

func TestSubmitCertificateVerification(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	certifierPubKey, certifierPrivKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := server.DefaultConfig()
	cfg.DisableWorker = true
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.PowDifficulty = 3
	cfg.Registration.Certifier = &registration.CertifierConfig{
		URL:    "http://localhost:8080",
		PubKey: registration.Base64Enc(certifierPubKey),
	}

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// User credentials
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	// Submit challenge with valid signature but invalid pow
	challenge := []byte("poet challenge")

	signature := ed25519.Sign(privKey, challenge)

	t.Run("invalid certificate", func(t *testing.T) {
		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: challenge,
			Pubkey:    pubKey,
			Signature: signature,
			Certificate: &api.SubmitRequest_Certificate{
				Signature: []byte("invalid signature"),
			},
		})
		require.ErrorIs(t, err, status.Error(codes.Unauthenticated, "invalid certificate\nsignature mismatch"))
	})
	t.Run("valid certificate", func(t *testing.T) {
		cert := &shared.Cert{
			Pubkey: pubKey,
		}
		certBytes, err := shared.EncodeCert(cert)
		require.NoError(t, err)

		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: challenge,
			Pubkey:    pubKey,
			Signature: signature,
			Certificate: &api.SubmitRequest_Certificate{
				Data:      certBytes,
				Signature: ed25519.Sign(certifierPrivKey, certBytes),
			},
		})
		require.NoError(t, err)
	})
	t.Run("no certificate - fallback to PoW (invalid)", func(t *testing.T) {
		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: challenge,
			Pubkey:    pubKey,
			Signature: signature,
		})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid proof of work parameters"))
	})
	t.Run("no certificate - fallback to PoW (valid)", func(t *testing.T) {
		resp, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
		require.NoError(t, err)
		nonce, err := shared.FindSubmitPowNonce(
			context.Background(),
			resp.PowParams.Challenge,
			challenge,
			pubKey,
			uint(resp.PowParams.Difficulty),
		)
		require.NoError(t, err)

		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Nonce:     nonce,
			Challenge: challenge,
			Pubkey:    pubKey,
			Signature: signature,
			PowParams: resp.PowParams,
		})
		require.NoError(t, err)
	})
	cancel()
	require.NoError(t, eg.Wait())
}

func createTestKeyFile(t *testing.T, dir, name string, data []byte) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}
	return path
}

// Test submitting a challenge followed by proof generation and getting the proof via GRPC.
func TestLoadTrustedKeysAndSubmit(t *testing.T) {
	const keysNum = 2

	// create test files
	dir := t.TempDir()
	t.Cleanup(func() { os.RemoveAll(dir) })

	var (
		trustedKeyCert *shared.OpaqueCert
		trustedKey     []byte
	)

	challenge := []byte("challenge")

	// User credentials
	userPubKey, userPrivKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	signature := ed25519.Sign(userPrivKey, challenge)

	for i := 0; i < keysNum; i++ {
		pubKey, private, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)

		createTestKeyFile(t, dir, fmt.Sprintf("valid_key_%d.key", i), []byte(base64.StdEncoding.EncodeToString(pubKey)))

		if i == keysNum-1 {
			expiration := time.Now().Add(time.Hour)
			data, err := shared.EncodeCert(&shared.Cert{Pubkey: userPubKey, Expiration: &expiration})
			require.NoError(t, err)

			trustedKeyCert = &shared.OpaqueCert{
				Data:      data,
				Signature: ed25519.Sign(private, data),
			}
			trustedKey = pubKey
		}
	}

	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	certifierPubKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := server.DefaultConfig()
	cfg.DisableWorker = true
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.PowDifficulty = 3
	cfg.Registration.Certifier = &registration.CertifierConfig{
		URL:                "http://localhost:8080",
		PubKey:             registration.Base64Enc(certifierPubKey),
		TrustedKeysDirPath: dir,
	}

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	conn, err := grpc.NewClient(
		srv.GrpcAddr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	configClient := api.NewConfigurationServiceClient(conn)

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})

	// trusted keys are not loaded
	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    userPubKey,
		Signature: signature,
		Certificate: &api.SubmitRequest_Certificate{
			Data:      trustedKeyCert.Data,
			Signature: trustedKeyCert.Signature,
		},
		CertificatePubkeyHint: trustedKey[:shared.CertPubkeyHintSize],
	})
	require.ErrorContains(t, err, registration.ErrInvalidCertificate.Error())

	// load trusted keys
	_, err = configClient.ReloadTrustedKeys(context.Background(), &api.ReloadTrustedKeysRequest{})
	require.NoError(t, err)

	_, err = client.Submit(context.Background(), &api.SubmitRequest{
		Challenge: challenge,
		Pubkey:    userPubKey,
		Signature: signature,
		Certificate: &api.SubmitRequest_Certificate{
			Data:      trustedKeyCert.Data,
			Signature: trustedKeyCert.Signature,
		},
		CertificatePubkeyHint: trustedKey[:shared.CertPubkeyHintSize],
	})
	require.NoError(t, err)
}

// Test submitting a challenge followed by proof generation and getting the proof via GRPC.
func TestSubmitAndGetProof(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Genesis = server.Genesis(time.Now().Add(time.Second))
	cfg.Round.EpochDuration = time.Second * 2
	cfg.Round.PhaseShift = 0
	cfg.Round.CycleGap = 0
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

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

	// single-element tree: the leaf is the root
	root := proof.Proof.Members[0]

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
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	cfg.Registration.MaxRoundMembers = 2

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
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

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

func TestSubmittingWithNeedByTimestamp(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	cfg.Round = &server.RoundConfig{
		EpochDuration: time.Hour,
		PhaseShift:    time.Minute * 10,
	}

	srv, client := spawnPoet(ctx, t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })

	var eg errgroup.Group
	eg.Go(func() error {
		return srv.Start(ctx)
	})
	t.Cleanup(func() { assert.NoError(t, eg.Wait()) })

	powParams, err := client.PowParams(context.Background(), &api.PowParamsRequest{})
	req.NoError(err)

	submitChallenge := func(ch []byte, deadline time.Time) error {
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		req.NoError(err)
		_, err = client.Submit(context.Background(), &api.SubmitRequest{
			Challenge: ch,
			Pubkey:    pubKey,
			PowParams: powParams.PowParams,
			Signature: ed25519.Sign(privKey, ch),
			Deadline:  timestamppb.New(deadline),
		})
		return err
	}

	round0End := cfg.Round.RoundEnd(cfg.Genesis.Time(), 0)

	req.NoError(submitChallenge([]byte("at round end"), round0End))
	req.NoError(submitChallenge([]byte("after round ends"), round0End.Add(time.Minute)))
	req.ErrorIs(
		submitChallenge([]byte("before round ends"), round0End.Add(-time.Minute)),
		status.Error(codes.FailedPrecondition, registration.ErrTooLateToRegister.Error()),
	)
}

func TestPersistingPowParams(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
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
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
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
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.Round.EpochDuration = time.Minute * 2
	cfg.Round.PhaseShift = time.Minute
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost
	server.SetupConfig(cfg)

	srv, err := server.New(context.Background(), *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })
	req.NoError(err)

	concurrentSubmits := 10000
	// spawn clients
	clients := make([]api.PoetServiceClient, 0, concurrentSubmits)
	for i := 0; i < concurrentSubmits; i++ {
		conn, err := grpc.NewClient(
			srv.GrpcAddr().String(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
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

// Test submitting a challenge followed by proof generation and getting the proof via GRPC
// in registration-only mode.
// It should return an empty proof but with members populated.
func TestRegistrationOnlyMode(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewContext(ctx, zaptest.NewLogger(t))

	cfg := server.DefaultConfig()
	cfg.DisableWorker = true
	cfg.PoetDir = t.TempDir()
	cfg.Genesis = server.Genesis(time.Now())
	cfg.Round.EpochDuration = time.Millisecond * 10
	cfg.Round.PhaseShift = 0
	cfg.Round.CycleGap = 0
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

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

	// Query for the proof
	var proof *api.ProofResponse
	req.Eventually(func() bool {
		proof, err = client.Proof(context.Background(), &api.ProofRequest{RoundId: resp.RoundId})
		return err == nil
	}, time.Second, time.Millisecond*10)

	req.Zero(proof.Proof.Leaves)
	req.Len(proof.Proof.Members, 1)
	req.Contains(proof.Proof.Members, challenge)
	cancel()

	merkleProof := shared.MerkleProof{
		Root:         proof.Proof.Proof.Root,
		ProvenLeaves: proof.Proof.Proof.ProvenLeaves,
		ProofNodes:   proof.Proof.Proof.ProofNodes,
	}

	// single-element tree: the leaf is the root
	root := proof.Proof.Members[0]
	labelHashFunc := hash.GenLabelHashFunc(root)
	merkleHashFunc := hash.GenMerkleHashFunc(root)
	req.Error(verifier.Validate(merkleProof, labelHashFunc, merkleHashFunc, proof.Proof.Leaves, shared.T))

	req.NoError(eg.Wait())
}

func TestConfiguringPrivateKey(t *testing.T) {
	cfg := server.DefaultConfig()
	cfg.DisableWorker = true
	cfg.PoetDir = t.TempDir()
	cfg.RawRPCListener = randomHost
	cfg.RawRESTListener = randomHost

	pubKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	err = os.Setenv(server.KeyEnvVar, base64.StdEncoding.EncodeToString(privateKey))
	require.NoError(t, err)

	srv := spawnPoetServer(context.Background(), t, *cfg)
	t.Cleanup(func() { assert.NoError(t, srv.Close()) })
	require.Equal(t, pubKey, srv.PublicKey())
}
