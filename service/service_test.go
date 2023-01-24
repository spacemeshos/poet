package service_test

import (
	"context"
	"crypto/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/gateway/challenge_verifier/mocks"
	"github.com/spacemeshos/poet/service"
)

type challenge struct {
	data   []byte
	nodeID []byte
}

func TestService_Recovery(t *testing.T) {
	req := require.New(t)
	cfg := &service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Second * 2,
		PhaseShift:    time.Second,
	}

	ctrl := gomock.NewController(t)
	verifier := mocks.NewMockVerifier(ctrl)
	tempdir := t.TempDir()

	// Generate groups of random challenges.
	challengeGroupSize := 5
	challengeGroups := make([][]challenge, 3)
	for i := 0; i < 3; i++ {
		challengeGroup := make([]challenge, challengeGroupSize)
		for i := 0; i < challengeGroupSize; i++ {
			challengeGroup[i] = challenge{data: make([]byte, 32), nodeID: make([]byte, 32)}
			_, err := rand.Read(challengeGroup[i].data)
			req.NoError(err)
			_, err = rand.Read(challengeGroup[i].nodeID)
			req.NoError(err)
		}
		challengeGroups[i] = challengeGroup
	}

	// Create a new service instance.
	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	submitChallenges := func(roundID string, challenges []challenge) {
		for _, challenge := range challenges {
			verifier.EXPECT().
				Verify(gomock.Any(), challenge.data, nil).
				Return(&challenge_verifier.Result{Hash: challenge.data, NodeId: challenge.nodeID}, nil)
			result, err := s.Submit(context.Background(), challenge.data, nil)
			req.NoError(err)
			req.Equal(challenge.data, result.Hash)
			req.Equal(roundID, result.Round)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background(), verifier))

	// Submit challenges to open round (0).
	submitChallenges("0", challengeGroups[0])

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return slices.Contains(info.ExecutingRoundsIds, "0")
	}, cfg.EpochDuration*2, time.Millisecond*100)

	// Submit challenges to open round (1).
	submitChallenges("1", challengeGroups[1])

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
	req.Len(info.ExecutingRoundsIds, 1)
	req.Contains(info.ExecutingRoundsIds, "0")
	req.Equal([]string{"0"}, info.ExecutingRoundsIds)

	req.NoError(s.Start(context.Background(), verifier))
	// Wait for round 2 to open
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.OpenRoundID == "2"
	}, cfg.EpochDuration*2, time.Millisecond*100)

	submitChallenges("2", challengeGroups[2])

	for i := 0; i < len(challengeGroups); i++ {
		proof := <-s.ProofsChan()
		req.Equal(strconv.Itoa(i), proof.RoundID)
		// Verify the submitted challenges.
		req.Len(proof.Members, len(challengeGroups[i]), "round: %v i: %d", proof.RoundID, i)
		for _, ch := range challengeGroups[i] {
			req.Contains(proof.Members, ch.data, "round: %v, i: %d", proof.RoundID, i)
		}
	}

	cancel()
	req.NoError(eg.Wait())
}

func TestNewService(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()

	cfg := new(service.Config)
	cfg.Genesis = time.Now().Add(time.Second).Format(time.RFC3339)
	cfg.EpochDuration = time.Second * 2
	cfg.PhaseShift = time.Second

	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)
	ctrl := gomock.NewController(t)
	verifier := mocks.NewMockVerifier(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background(), verifier))

	challengesCount := 8
	challenges := make([]challenge, challengesCount)

	// Generate random challenges.
	for i := 0; i < len(challenges); i++ {
		challenges[i] = challenge{data: make([]byte, 32), nodeID: make([]byte, 32)}
		_, err := rand.Read(challenges[i].data)
		req.NoError(err)
		_, err = rand.Read(challenges[i].nodeID)
		req.NoError(err)
	}

	info, err := s.Info(context.Background())
	require.NoError(t, err)
	currentRound := info.OpenRoundID

	// Submit challenges.
	for i := 0; i < len(challenges); i++ {
		verifier.EXPECT().
			Verify(gomock.Any(), challenges[i].data, nil).
			Return(&challenge_verifier.Result{Hash: challenges[i].data, NodeId: challenges[i].nodeID}, nil)
		result, err := s.Submit(context.Background(), challenges[i].data, nil)
		req.NoError(err)
		req.Equal(challenges[i].data, result.Hash)
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
		for _, r := range info.ExecutingRoundsIds {
			if r == currentRound {
				return true
			}
		}
		return false
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
	proof := <-s.ProofsChan()

	req.Equal(currentRound, proof.RoundID)
	// Verify the submitted challenges.
	req.Len(proof.Members, len(challenges))
	for _, ch := range challenges {
		req.Contains(proof.Members, ch.data)
	}

	cancel()
	req.NoError(eg.Wait())
}

func TestSubmitIdempotency(t *testing.T) {
	req := require.New(t)
	cfg := service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Hour,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}
	challenge := []byte("challenge")
	signature := []byte("signature")

	s, err := service.NewService(context.Background(), &cfg, t.TempDir())
	req.NoError(err)

	verifier := mocks.NewMockVerifier(gomock.NewController(t))
	verifier.EXPECT().
		Verify(gomock.Any(), challenge, signature).
		Times(2).
		Return(&challenge_verifier.Result{Hash: []byte("hash")}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background(), verifier))

	// Submit challenge
	result, err := s.Submit(context.Background(), challenge, signature)
	req.NoError(err)
	req.Equal(result.Hash, []byte("hash"))

	// Try again - it should return the same result
	result, err = s.Submit(context.Background(), challenge, signature)
	req.NoError(err)
	req.Equal(result.Hash, []byte("hash"))

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
				Genesis:       time.Now().Add(time.Minute).Format(time.RFC3339),
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
		req.Empty(info.ExecutingRoundsIds)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       time.Now().Add(-time.Minute).Format(time.RFC3339),
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
		req.Empty(info.ExecutingRoundsIds)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("in first epoch", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       time.Now().Add(-time.Minute).Format(time.RFC3339),
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
		req.Empty(info.ExecutingRoundsIds)

		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("in distant epoch", func(t *testing.T) {
		t.Parallel()
		s, err := service.NewService(
			context.Background(),
			&service.Config{
				Genesis:       time.Now().Add(-time.Hour * 100).Format(time.RFC3339),
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
		req.Empty(info.ExecutingRoundsIds)

		cancel()
		req.NoError(eg.Wait())
	})
}

func TestService_Start(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	cfg := &service.Config{
		Genesis:       time.Now().Add(time.Minute).Format(time.RFC3339),
		EpochDuration: time.Hour,
	}
	t.Run("cannot start twice", func(t *testing.T) {
		s, err := service.NewService(context.Background(), cfg, t.TempDir())
		req.NoError(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var eg errgroup.Group
		eg.Go(func() error { return s.Run(ctx) })
		req.NoError(s.Start(context.Background(), mocks.NewMockVerifier(gomock.NewController(t))))
		req.ErrorIs(s.Start(context.Background(), mocks.NewMockVerifier(gomock.NewController(t))), service.ErrAlreadyStarted)
		cancel()
		req.NoError(eg.Wait())
	})
	t.Run("hang start respects context cancellation", func(t *testing.T) {
		s, err := service.NewService(context.Background(), cfg, t.TempDir())
		req.NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		req.ErrorIs(s.Start(ctx, mocks.NewMockVerifier(gomock.NewController(t))), context.DeadlineExceeded)
	})
}

func TestService_Recovery_MissingOpenRound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cfg := &service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Hour,
		PhaseShift:    time.Second,
	}

	ctrl := gomock.NewController(t)
	challengeVerifier := mocks.NewMockVerifier(ctrl)
	tempdir := t.TempDir()

	// Create a new service instance.
	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background(), challengeVerifier))

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return slices.Contains(info.ExecutingRoundsIds, "0")
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
	req.Len(info.ExecutingRoundsIds, 1)
	req.Contains(info.ExecutingRoundsIds, "0")
	req.Equal([]string{"0"}, info.ExecutingRoundsIds)

	cancel()
	req.NoError(eg.Wait())
}

// Test if `reset` flag works.
// Scenario:
// - run Service and wait for round 0 to start executing,
// - shutdown the Service
// - restart Service with reset flag == true
// - expect the persisted state to be discarded:
//   - round 0 should not exist
//   - round 1 should be open
func TestService_Recovery_Reset(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	cfg := &service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Hour,
		Reset:         true,
	}
	ctrl := gomock.NewController(t)
	challengeVerifier := mocks.NewMockVerifier(ctrl)
	tempdir := t.TempDir()

	// Create a new service instance.
	s, err := service.NewService(context.Background(), cfg, tempdir)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })
	req.NoError(s.Start(context.Background(), challengeVerifier))

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return slices.Contains(info.ExecutingRoundsIds, "0")
	}, cfg.EpochDuration*2, time.Millisecond*100)

	cancel()
	req.NoError(eg.Wait())

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

	// Service instance should not recover the round 0 in-execution.
	info, err := s.Info(ctx)
	req.NoError(err)
	req.Equal("1", info.OpenRoundID)
	req.Empty(info.ExecutingRoundsIds)

	cancel()
	req.NoError(eg.Wait())
}
