package service_test

import (
	"bytes"
	"context"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

func TestService_Recovery(t *testing.T) {
	req := require.New(t)
	cfg := &service.Config{
		Genesis:         service.Genesis(time.Now().Add(time.Second)),
		EpochDuration:   time.Second * 5,
		PhaseShift:      time.Second * 2,
		MaxRoundMembers: 100,
	}
	tempdir := t.TempDir()

	// Generate groups of random challenges.
	challengeGroupSize := 5
	challengeGroups := make([][]challenge, 3)
	for g := 0; g < 3; g++ {
		challengeGroup := make([]challenge, challengeGroupSize)
		for i := 0; i < challengeGroupSize; i++ {
			challengeGroup[i] = challenge{
				data:   bytes.Repeat([]byte{byte(g*10 + i)}, 32),
				nodeID: bytes.Repeat([]byte{byte(-g*10 - i)}, 32),
			}
		}
		challengeGroups[g] = challengeGroup
	}

	submitChallenges := func(s *service.Service, roundID string, challenges []challenge) {
		params := s.PowParams()
		for _, challenge := range challenges {
			nonce, _ := shared.FindSubmitPowNonce(
				context.Background(),
				params.Challenge,
				challenge.data,
				challenge.nodeID,
				params.Difficulty,
			)
			result, err := s.Submit(context.Background(), challenge.data, challenge.nodeID, nonce, params)
			req.NoError(err)
			req.Equal(roundID, result.Round)
		}
	}

	// Create a new service instance.
	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background()))

	// Submit challenges to open round (0).
	submitChallenges(s, "0", challengeGroups[0])

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.ExecutingRoundId != nil && *info.ExecutingRoundId == "0"
	}, cfg.EpochDuration*2, time.Millisecond*100)

	// Submit challenges to open round (1).
	submitChallenges(s, "1", challengeGroups[1])

	cancel()
	req.NoError(eg.Wait())

	// Create a new service instance.
	s, err = service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	eg = errgroup.Group{}
	eg.Go(func() error { return s.Run(ctx) })

	// Service instance should recover 2 rounds: round 0 in executing state, and round 1 in open state.
	info, err := s.Info(context.Background())
	req.NoError(err)
	req.Equal("1", info.OpenRoundID)
	req.Equal("0", *info.ExecutingRoundId)

	req.NoError(s.Start(context.Background()))
	// Wait for round 2 to open
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.OpenRoundID == "2" && info.ExecutingRoundId != nil && *info.ExecutingRoundId == "1"
	}, cfg.EpochDuration*2, time.Millisecond*5)

	submitChallenges(s, "2", challengeGroups[2])

	for i := 0; i < len(challengeGroups); i++ {
		proofMsg := <-s.ProofsChan()
		proof := proofMsg.Proof
		req.Equal(strconv.Itoa(i), proofMsg.RoundID)
		// Verify the submitted challenges.
		req.Len(proof.Members, len(challengeGroups[i]), "round: %v i: %d", proofMsg.RoundID, i)
		for _, ch := range challengeGroups[i] {
			req.Contains(proof.Members, ch.data, "round: %v, i: %d", proofMsg.RoundID, i)
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
	}

	cancel()
	req.NoError(eg.Wait())
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

	s, err := service.NewService(context.Background(), &cfg, tempdir, service.WithPowVerifier(powVerifier))
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

	info, err := s.Info(context.Background())
	require.NoError(t, err)
	currentRound := info.OpenRoundID

	// Submit challenges.
	for _, ch := range challenges {
		powVerifier.EXPECT().Verify(ch.data, ch.nodeID, uint64(0)).Return(nil)
		result, err := s.Submit(context.Background(), ch.data, ch.nodeID, 0, service.PowParams{})
		req.NoError(err)
		req.Equal(currentRound, result.Round)
	}

	// Verify that round is still open.
	info, err = s.Info(context.Background())
	req.NoError(err)
	req.Equal(currentRound, info.OpenRoundID)

	// Wait for round to start execution.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.ExecutingRoundId != nil && *info.ExecutingRoundId == currentRound
	}, cfg.EpochDuration*2, time.Millisecond*100)

	// Wait for end of execution.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		prevRoundID, err := strconv.Atoi(currentRound)
		req.NoError(err)
		currRoundID, err := strconv.Atoi(info.OpenRoundID)
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

	s, err := service.NewService(context.Background(), &cfg, t.TempDir(), service.WithPowVerifier(verifier))
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

func TestService_OpeningRounds(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	t.Run("before genesis", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       service.Genesis(time.Now().Add(time.Minute)),
				EpochDuration: time.Hour,
			},
			t.TempDir(),
		)
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })

		// Service instance should create open round 0.
		info, err := s.Info(ctx)
		req.NoError(err)
		req.Equal("0", info.OpenRoundID)
		req.Nil(info.ExecutingRoundId)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       service.Genesis(time.Now().Add(-time.Minute)),
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute * 10,
			},
			t.TempDir(),
		)
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })

		// Service instance should create open round 0.
		info, err := s.Info(ctx)
		req.NoError(err)
		req.Equal("0", info.OpenRoundID)
		req.Nil(info.ExecutingRoundId)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("in first epoch", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       service.Genesis(time.Now().Add(-time.Minute)),
				EpochDuration: time.Hour,
			},
			t.TempDir(),
		)
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })

		// Service instance should create open round 1.
		info, err := s.Info(ctx)
		req.NoError(err)
		req.Equal("1", info.OpenRoundID)
		req.Empty(info.ExecutingRoundId)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("in distant epoch", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       service.Genesis(time.Now().Add(-time.Hour * 100)),
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute,
			},
			t.TempDir(),
		)
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })

		// Service instance should create open round 1.
		info, err := s.Info(ctx)
		req.NoError(err)
		req.Equal("100", info.OpenRoundID)
		req.Empty(info.ExecutingRoundId)

		cancel()
		req.NoError(eg.Wait())
	})
}

func TestService_Start(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	cfg := &service.Config{
		Genesis:       service.Genesis(time.Now().Add(time.Minute)),
		EpochDuration: time.Hour,
	}
	t.Run("cannot start twice", func(t *testing.T) {
		s, err := service.NewService(context.Background(), cfg, t.TempDir())
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
		s, err := service.NewService(context.Background(), cfg, t.TempDir())
		req.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		req.ErrorIs(s.Start(ctx), context.DeadlineExceeded)
	})
}

func TestService_Recovery_MissingOpenRound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cfg := &service.Config{
		Genesis:       service.Genesis(time.Now().Add(time.Second)),
		EpochDuration: time.Hour,
		PhaseShift:    time.Second,
	}

	tempdir := t.TempDir()

	// Create a new service instance.
	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background()))

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.ExecutingRoundId != nil && *info.ExecutingRoundId == "0"
	}, cfg.EpochDuration*2, time.Millisecond*100)

	cancel()
	req.NoError(eg.Wait())

	req.NoError(os.RemoveAll(path.Join(tempdir, "rounds", "1")))
	time.Sleep(time.Second)

	// Create a new service instance.
	s, err = service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	eg = errgroup.Group{}
	eg.Go(func() error {
		err := s.Run(ctx)
		assert.NoError(t, err)
		cancel()
		return err
	})

	// Service instance should recover 2 rounds:
	// - round 0 in executing state (from files),
	// - round 1 in open state - it should be recreated.
	info, err := s.Info(ctx)
	req.NoError(err)
	req.Equal("1", info.OpenRoundID)
	req.Equal("0", *info.ExecutingRoundId)

	cancel()
	req.NoError(eg.Wait())
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

	s, err := service.NewService(context.Background(), &cfg, t.TempDir())
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
