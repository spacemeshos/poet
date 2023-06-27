package service_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/service/mocks"
)

func TestRegistration_OpeningRounds(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	cfg := &service.Config{
		Genesis:       service.Genesis(time.Now()),
		EpochDuration: time.Hour,
		PhaseShift:    time.Minute * 10,
	}

	t.Run("before genesis", func(t *testing.T) {
		t.Parallel()
		clock := mocks.NewMockClock(gomock.NewController(t))
		clock.EXPECT().Now().Return(cfg.Genesis.Time().Add(-time.Minute))
		s, err := service.NewRegistry(
			context.Background(),
			cfg,
			clock,
			t.TempDir(),
		)
		req.NoError(err)

		req.Equal("0", s.OpenRound().ID)
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		t.Parallel()
		clock := mocks.NewMockClock(gomock.NewController(t))
		clock.EXPECT().Now().Return(cfg.Genesis.Time().Add(time.Minute))
		s, err := service.NewRegistry(
			context.Background(),
			cfg,
			clock,
			t.TempDir(),
		)
		req.NoError(err)

		req.Equal("0", s.OpenRound().ID)
	})
	t.Run("in first epoch", func(t *testing.T) {
		t.Parallel()
		clock := mocks.NewMockClock(gomock.NewController(t))
		clock.EXPECT().Now().Return(cfg.Genesis.Time().Add(cfg.EpochDuration))
		s, err := service.NewRegistry(
			context.Background(),
			cfg,
			clock,
			t.TempDir(),
		)
		req.NoError(err)

		req.Equal("1", s.OpenRound().ID)
	})
	t.Run("in distant epoch", func(t *testing.T) {
		t.Parallel()
		clock := mocks.NewMockClock(gomock.NewController(t))
		clock.EXPECT().Now().Return(cfg.Genesis.Time().Add(cfg.EpochDuration * 100))
		s, err := service.NewRegistry(
			context.Background(),
			cfg,
			clock,
			t.TempDir(),
		)
		req.NoError(err)

		req.Equal("100", s.OpenRound().ID)
	})
}

func TestRegistration_Recovery(t *testing.T) {
	req := require.New(t)
	cfg := &service.Config{
		Genesis:         service.Genesis(time.Now()),
		EpochDuration:   time.Second * 5,
		PhaseShift:      time.Second * 2,
		MaxRoundMembers: 100,
	}
	tempdir := t.TempDir()
	clock := mocks.NewMockClock(gomock.NewController(t))
	clock.EXPECT().Now().Return(cfg.Genesis.Time())

	// Generate groups of random challenges.
	challengeGroupSize := byte(5)
	challengeGroups := make([][]challenge, 3)
	for g := byte(0); g < 3; g++ {
		challengeGroup := make([]challenge, challengeGroupSize)
		for i := byte(0); i < challengeGroupSize; i++ {
			challengeGroup[i] = challenge{
				data:   bytes.Repeat([]byte{g*10 + i}, 32),
				nodeID: bytes.Repeat([]byte{-g*10 - i}, 32),
			}
		}
		challengeGroups[g] = challengeGroup
	}

	reg, err := service.NewRegistry(context.Background(), cfg, clock, tempdir)
	req.NoError(err)

	submitChallenges := func(roundID string, challenges []challenge) {
		for _, challenge := range challenges {
			result, err := reg.Register(context.Background(), challenge.data, challenge.nodeID)
			req.NoError(err)
			req.Equal(roundID, result.Round)
		}
	}

	submitChallenges("0", challengeGroups[0])
	_, err = reg.CloseOpenRound(context.Background())
	req.NoError(err)

	submitChallenges("1", challengeGroups[1])

	req.NoError(reg.Close())

	// Create a new instance instance.
	reg, err = service.NewRegistry(context.Background(), cfg, clock, tempdir)
	req.NoError(err)

	// It should recover 2 rounds: round 0 in executing state, and round 1 in open state.
	open := reg.OpenRound()
	req.NotNil(open)
	req.Equal("1", open.ID)
	req.Equal(uint(1), open.Epoch())

	executing := reg.ExecutingRound()
	req.NotNil(executing)
	req.Equal("0", executing.ID)
	req.Equal(0, executing.Epoch())

	req.NoError(reg.Close())
}
