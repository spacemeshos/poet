package service_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spacemeshos/go-scale"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/gateway/challenge_verifier/mocks"
	"github.com/spacemeshos/poet/service"
)

type MockBroadcaster struct {
	receivedMessages chan []byte
}

func (b *MockBroadcaster) BroadcastProof(msg []byte, roundID string, members [][]byte) error {
	b.receivedMessages <- msg
	return nil
}

type challenge struct {
	data   []byte
	nodeID []byte
}

func TestService_Recovery(t *testing.T) {
	req := require.New(t)
	broadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	cfg := &service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Second,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}

	ctrl := gomock.NewController(t)
	verifier := mocks.NewMockVerifier(ctrl)

	tempdir := t.TempDir()

	// Create a new service instance.
	s, err := service.NewService(cfg, tempdir)
	req.NoError(err)
	err = s.Start(broadcaster, verifier)
	req.NoError(err)

	// Generate 4 groups of random challenges.
	challengeGroupSize := 10
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

	submitChallenges := func(roundID string, challenges []challenge) {
		for _, challenge := range challenges {
			verifier.EXPECT().Verify(gomock.Any(), challenge.data, nil).Return(&challenge_verifier.Result{Hash: challenge.data, NodeId: challenge.nodeID}, nil)
			round, hash, err := s.Submit(context.Background(), challenge.data, nil)
			req.NoError(err)
			req.Equal(challenge.data, hash)
			req.Equal(roundID, round)
		}
	}

	// Submit challenges to open round (0).
	submitChallenges("0", challengeGroups[0])

	// Wait for round 0 to start executing.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return slices.Contains(info.ExecutingRoundsIds, "0")
	}, cfg.EpochDuration*2, time.Millisecond*20)

	// Submit challenges to open round (1).
	submitChallenges("1", challengeGroups[1])

	req.NoError(s.Shutdown())

	// Create a new service instance.
	s, err = service.NewService(cfg, tempdir)
	req.NoError(err)

	err = s.Start(broadcaster, verifier)
	req.NoError(err)

	// Service instance should recover 2 rounds: round 0 in executing state, and round 1 in open state.
	info, err := s.Info(context.Background())
	req.NoError(err)
	req.Equal("1", info.OpenRoundID)
	req.Len(info.ExecutingRoundsIds, 1)
	req.Contains(info.ExecutingRoundsIds, "0")
	req.Equal([]string{"0"}, info.ExecutingRoundsIds)

	// Wait for round 2 to open
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		return info.OpenRoundID == "2"
	}, cfg.EpochDuration*2, time.Millisecond*20)

	submitChallenges("2", challengeGroups[2])

	for i := 0; i < len(challengeGroups); i++ {
		msg := <-broadcaster.receivedMessages
		proofMsg := service.PoetProofMessage{}
		dec := scale.NewDecoder(bytes.NewReader(msg))
		_, err := proofMsg.DecodeScale(dec)
		req.NoError(err)

		req.Equal(strconv.Itoa(i), proofMsg.RoundID)
		// Verify the submitted challenges.
		req.Len(proofMsg.Members, len(challengeGroups[i]), "round: %v i: %d", proofMsg.RoundID, i)
		for _, ch := range challengeGroups[i] {
			req.Contains(proofMsg.Members, ch.data, "round: %v, i: %d", proofMsg.RoundID, i)
		}
	}

	req.NoError(s.Shutdown())
}

func TestConcurrentServiceStartAndShutdown(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	cfg := service.Config{
		Genesis:       time.Now().Add(2 * time.Second).Format(time.RFC3339),
		EpochDuration: time.Second,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}
	ctrl := gomock.NewController(t)
	verifier := mocks.NewMockVerifier(ctrl)

	for i := 0; i < 100; i += 1 {
		t.Run(fmt.Sprintf("iteration %d", i), func(t *testing.T) {
			t.Parallel()
			s, err := service.NewService(&cfg, t.TempDir())
			req.NoError(err)

			var eg errgroup.Group
			eg.Go(func() error {
				proofBroadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
				req.NoError(s.Start(proofBroadcaster, verifier))
				return nil
			})
			req.Eventually(func() bool { return s.Shutdown() == nil }, time.Second, time.Millisecond*10)
			eg.Wait()
		})
	}
}

func TestNewService(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()

	cfg := new(service.Config)
	cfg.Genesis = time.Now().Add(time.Second).Format(time.RFC3339)
	cfg.EpochDuration = time.Second
	cfg.PhaseShift = time.Second / 2
	cfg.CycleGap = time.Second / 4

	s, err := service.NewService(cfg, tempdir)
	req.NoError(err)
	proofBroadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	ctrl := gomock.NewController(t)
	verifier := mocks.NewMockVerifier(ctrl)
	err = s.Start(proofBroadcaster, verifier)
	req.NoError(err)

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
		verifier.EXPECT().Verify(gomock.Any(), challenges[i].data, nil).Return(&challenge_verifier.Result{Hash: challenges[i].data, NodeId: challenges[i].nodeID}, nil)
		round, hash, err := s.Submit(context.Background(), challenges[i].data, nil)
		req.NoError(err)
		req.Equal(challenges[i].data, hash)
		req.Equal(currentRound, round)
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
	}, cfg.EpochDuration*2, time.Millisecond*20)

	// Wait for end of execution.
	req.Eventually(func() bool {
		info, err := s.Info(context.Background())
		req.NoError(err)
		prevRoundID, err := strconv.Atoi(currentRound)
		req.NoError(err)
		currRoundID, err := strconv.Atoi(info.OpenRoundID)
		req.NoError(err)
		return currRoundID >= prevRoundID+1
	}, time.Second, time.Millisecond*20)

	// Wait for proof message broadcast.
	msg := <-proofBroadcaster.receivedMessages
	proof := service.PoetProofMessage{}
	dec := scale.NewDecoder(bytes.NewReader(msg))
	_, err = proof.DecodeScale(dec)
	req.NoError(err)

	req.Equal(currentRound, proof.RoundID)
	// Verify the submitted challenges.
	req.Len(proof.Members, len(challenges))
	for _, ch := range challenges {
		req.Contains(proof.Members, ch.data)
	}

	req.NoError(s.Shutdown())
}

func TestSubmitIdempotency(t *testing.T) {
	req := require.New(t)
	cfg := service.Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Second,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}
	challenge := []byte("challenge")
	signature := []byte("signature")

	s, err := service.NewService(&cfg, t.TempDir())
	req.NoError(err)

	proofBroadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	verifier := mocks.NewMockVerifier(gomock.NewController(t))
	verifier.EXPECT().Verify(gomock.Any(), challenge, signature).Times(2).Return(&challenge_verifier.Result{Hash: []byte("hash")}, nil)
	err = s.Start(proofBroadcaster, verifier)
	req.NoError(err)

	// Submit challenge
	_, hash, err := s.Submit(context.Background(), challenge, signature)
	req.NoError(err)
	req.Equal(hash, []byte("hash"))

	// Try again - it should return the same result
	_, hash, err = s.Submit(context.Background(), challenge, signature)
	req.NoError(err)
	req.Equal(hash, []byte("hash"))

	req.NoError(s.Shutdown())
}
