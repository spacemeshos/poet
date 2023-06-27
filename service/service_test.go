package service_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/service/mocks"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

func TestParsingGenesisTime(t *testing.T) {
	t.Parallel()

	timeStr := "2006-01-02T15:04:05Z"
	var genesis service.Genesis
	err := genesis.UnmarshalFlag(timeStr)
	require.NoError(t, err)

	expected, err := time.Parse(time.RFC3339, timeStr)
	require.NoError(t, err)

	require.Equal(t, expected, genesis.Time())
}

type challenge struct {
	data   []byte
	nodeID []byte
}

func TestNewService(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()

	cfg := service.Config{
		Genesis:         service.Genesis(time.Now().Add(time.Second)),
		EpochDuration:   time.Second * 2,
		PhaseShift:      time.Second,
		MaxRoundMembers: 100,
	}

	ctrl := gomock.NewController(t)
	powVerifier := mocks.NewMockPowVerifier(ctrl)
	powVerifier.EXPECT().Params().AnyTimes().Return(service.PowParams{})
	powVerifier.EXPECT().SetParams(gomock.Any()).AnyTimes()

	reg, err := service.NewRegistry(context.Background(), &cfg, service.RealClock{}, tempdir)
	req.NoError(err)

	s, err := service.NewService(context.Background(), &cfg, tempdir, reg, service.WithPowVerifier(powVerifier))
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background()))

	challengesCount := 8
	challenges := make([]challenge, challengesCount)

	// Generate random challenges.
	for i := 0; i < len(challenges); i++ {
		challenges[i] = challenge{data: bytes.Repeat([]byte{byte(i)}, 32), nodeID: bytes.Repeat([]byte{-byte(i)}, 32)}
	}

	currentRound := reg.OpenRound().ID

	// Submit challenges.
	for _, ch := range challenges {
		powVerifier.EXPECT().Verify(ch.data, ch.nodeID, uint64(0)).Return(nil)
		result, err := s.Submit(context.Background(), ch.data, ch.nodeID, 0, service.PowParams{})
		req.NoError(err)
		req.Equal(currentRound, result.Round)
	}

	// Verify that round is still open.
	req.Equal(currentRound, reg.OpenRound().ID)

	// Wait for round to start execution.
	req.Eventually(func() bool {
		executing := reg.ExecutingRound()
		return executing != nil && executing.ID == currentRound
	}, cfg.EpochDuration*2, time.Millisecond*100)

	// Wait for end of execution.
	req.Eventually(func() bool {
		prevRoundID, err := strconv.Atoi(currentRound)
		req.NoError(err)
		currRoundID, err := strconv.Atoi(reg.OpenRound().ID)
		req.NoError(err)
		return currRoundID >= prevRoundID+1
	}, time.Second, time.Millisecond*100)

	// Wait for proof message.
	proofMsg := <-s.ProofsChan()
	proof := proofMsg.Proof

	req.Equal(currentRound, proofMsg.RoundID)
	// Verify the submitted challenges.
	req.Len(proof.Members, len(challenges))
	for _, ch := range challenges {
		req.Contains(proof.Members, ch.data)
	}

	challenge, err := prover.CalcTreeRoot(proof.Members)
	req.NoError(err)

	err = verifier.Validate(
		proof.MerkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		proof.NumLeaves,
		shared.T,
	)
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
}

func TestNewServiceCannotSetNilVerifier(t *testing.T) {
	t.Parallel()
	tempdir := t.TempDir()

	_, err := service.NewService(
		context.Background(),
		config.DefaultConfig().Service,
		tempdir,
		nil,
		service.WithPowVerifier(nil),
	)
	require.ErrorContains(t, err, "pow verifier cannot be nil")
}

func TestSubmitIdempotency(t *testing.T) {
	req := require.New(t)
	cfg := service.Config{
		Genesis:         service.Genesis(time.Now().Add(time.Second)),
		EpochDuration:   time.Hour,
		PhaseShift:      time.Second / 2,
		CycleGap:        time.Second / 4,
		MaxRoundMembers: 100,
	}
	challenge := []byte("challenge")
	nodeID := []byte("nodeID")
	nonce := uint64(7)

	verifier := mocks.NewMockPowVerifier(gomock.NewController(t))

	reg, err := service.NewRegistry(context.Background(), &cfg, service.RealClock{}, t.TempDir())
	req.NoError(err)

	s, err := service.NewService(context.Background(), &cfg, t.TempDir(), reg, service.WithPowVerifier(verifier))
	req.NoError(err)

	verifier.EXPECT().Params().Times(2).Return(service.PowParams{})
	verifier.EXPECT().Verify(challenge, nodeID, nonce).Times(2).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background()))

	// Submit challenge
	_, err = s.Submit(context.Background(), challenge, nodeID, nonce, service.PowParams{})
	req.NoError(err)

	// Try again - it should return the same result
	_, err = s.Submit(context.Background(), challenge, nodeID, nonce, service.PowParams{})
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
}

func TestService_Start(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	cfg := &service.Config{
		Genesis:       service.Genesis(time.Now().Add(time.Minute)),
		EpochDuration: time.Hour,
	}
	reg, err := service.NewRegistry(context.Background(), cfg, service.RealClock{}, t.TempDir())
	req.NoError(err)

	t.Run("cannot start twice", func(t *testing.T) {
		s, err := service.NewService(context.Background(), cfg, t.TempDir(), reg)
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })
		req.NoError(s.Start(context.Background()))
		req.ErrorIs(s.Start(context.Background()), service.ErrAlreadyStarted)
		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("hang start respects context cancellation", func(t *testing.T) {
		s, err := service.NewService(context.Background(), cfg, t.TempDir(), reg)
		req.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		req.ErrorIs(s.Start(ctx), context.DeadlineExceeded)
	})
}

// Test if Proof of Work challenge is rotated every round.
// The challenge should be changed to the root of PoET proof Merkle tree
// of the previous round.
func TestService_PowChallengeRotation(t *testing.T) {
	req := require.New(t)
	cfg := service.Config{
		Genesis:             service.Genesis(time.Now()),
		EpochDuration:       time.Second,
		PhaseShift:          time.Second / 2,
		InitialPowChallenge: "initial challenge",
		PowDifficulty:       7,
	}

	reg, err := service.NewRegistry(context.Background(), &cfg, service.RealClock{}, t.TempDir())
	req.NoError(err)

	s, err := service.NewService(context.Background(), &cfg, t.TempDir(), reg)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background()))

	params := s.PowParams()
	req.EqualValues(cfg.InitialPowChallenge, params.Challenge)
	req.EqualValues(cfg.PowDifficulty, params.Difficulty)

	proof := <-s.ProofsChan()
	params = s.PowParams()
	req.EqualValues(proof.Proof.Root, params.Challenge)
	req.EqualValues(cfg.PowDifficulty, params.Difficulty)

	cancel()
	req.NoError(eg.Wait())
}
