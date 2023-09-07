package registration_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/config/round_config"
	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/registration/mocks"
	"github.com/spacemeshos/poet/shared"
)

func TestSubmitIdempotence(t *testing.T) {
	req := require.New(t)
	genesis := time.Now().Add(time.Second)

	roundCfg := round_config.Config{
		EpochDuration: time.Hour,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}

	challenge := []byte("challenge")
	nodeID := []byte("nodeID")
	nonce := uint64(7)

	verifier := mocks.NewMockPowVerifier(gomock.NewController(t))
	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))

	s, err := registration.NewRegistration(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		registration.WithPowVerifier(verifier),
		registration.WithConfig(registration.Config{
			MaxRoundMembers: 100,
		}),
		registration.WithRoundConfig(roundCfg),
	)
	req.NoError(err)

	verifier.EXPECT().Params().Times(2).Return(registration.PowParams{})
	verifier.EXPECT().Verify(challenge, nodeID, nonce).Times(2).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	// Submit challenge
	epoch, _, err := s.Submit(context.Background(), challenge, nodeID, nonce, registration.PowParams{})
	req.NoError(err)
	req.Equal(uint(0), epoch)

	// Try again - it should return the same result
	epoch, _, err = s.Submit(context.Background(), challenge, nodeID, nonce, registration.PowParams{})
	req.NoError(err)
	req.Equal(uint(0), epoch)

	cancel()
	req.NoError(eg.Wait())
}

func TestService_OpeningRounds(t *testing.T) {
	t.Run("before genesis", func(t *testing.T) {
		reg, err := registration.NewRegistration(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
		)
		require.NoError(t, err)

		// Service instance should create open round 0.
		open, executing := reg.Info(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint(0), open)
		require.Nil(t, executing)
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		reg, err := registration.NewRegistration(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
			registration.WithRoundConfig(round_config.Config{
				PhaseShift: time.Minute * 10,
			}),
		)
		require.NoError(t, err)

		// Service instance should create open round 0.
		open, executing := reg.Info(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint(0), open)
		require.Nil(t, executing)
	})
	t.Run("in first epoch", func(t *testing.T) {
		reg, err := registration.NewRegistration(
			context.Background(),
			time.Now().Add(-time.Hour),
			t.TempDir(),
			nil,
			registration.WithRoundConfig(round_config.Config{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute * 10,
			}),
		)
		require.NoError(t, err)

		// Service instance should create open round 1.
		open, executing := reg.Info(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint(1), open)
		require.NotNil(t, executing)
		require.Equal(t, uint(0), *executing)
	})
	t.Run("in distant epoch", func(t *testing.T) {
		reg, err := registration.NewRegistration(
			context.Background(),
			time.Now().Add(-100*time.Hour),
			t.TempDir(),
			nil,
			registration.WithRoundConfig(round_config.Config{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute,
			}),
		)
		require.NoError(t, err)

		// Service instance should create open round 1.
		open, executing := reg.Info(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint(100), open)
		require.NotNil(t, executing)
		require.Equal(t, uint(99), *executing)
	})
}

// Test if Proof of Work challenge is rotated every round.
// The challenge should be changed to the root of PoET proof Merkle tree
// of the previous round.
func TestService_PowChallengeRotation(t *testing.T) {
	genesis := time.Now()

	proofs := make(chan shared.NIP, 1)

	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(proofs)
	workerSvc.EXPECT().
		ExecuteRound(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, epoch uint, _ []byte) error {
			proofs <- shared.NIP{
				MerkleProof: shared.MerkleProof{
					Root: []byte{1, 2, 3, 4},
				},
				Epoch: epoch,
			}
			return nil
		}).
		AnyTimes()

	reg, err := registration.NewRegistration(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		registration.WithRoundConfig(round_config.Config{EpochDuration: 10 * time.Millisecond}),
	)
	require.NoError(t, err)

	params0 := reg.PowParams()
	require.NotEqual(t, []byte{1, 2, 3, 4}, params0.Challenge)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return reg.Run(ctx) })

	require.Eventually(t, func() bool {
		return bytes.Equal([]byte{1, 2, 3, 4}, reg.PowParams().Challenge)
	}, time.Second, time.Millisecond)

	cancel()
	require.NoError(t, eg.Wait())
}
