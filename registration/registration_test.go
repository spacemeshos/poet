package registration_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/registration/mocks"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
)

func TestSubmitIdempotence(t *testing.T) {
	req := require.New(t)
	genesis := time.Now().Add(time.Second)

	roundCfg := server.RoundConfig{
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

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&roundCfg,
		registration.WithPowVerifier(verifier),
	)
	req.NoError(err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	verifier.EXPECT().Params().Times(2).Return(registration.PowParams{})
	verifier.EXPECT().Verify(challenge, nodeID, nonce).Times(2).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	// Submit challenge
	epoch, _, err := r.Submit(context.Background(), challenge, nodeID, nonce, registration.PowParams{}, time.Time{})
	req.NoError(err)
	req.Equal(uint(0), epoch)

	// Try again - it should return the same result
	epoch, _, err = r.Submit(context.Background(), challenge, nodeID, nonce, registration.PowParams{}, time.Time{})
	req.NoError(err)
	req.Equal(uint(0), epoch)

	cancel()
	req.NoError(eg.Wait())
}

func TestOpeningRounds(t *testing.T) {
	t.Parallel()
	t.Run("before genesis", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 0.
		require.Equal(t, uint(0), reg.OpenRound())
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{PhaseShift: time.Minute * 10},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 0.
		require.Equal(t, uint(0), reg.OpenRound())
	})
	t.Run("in first epoch", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(-time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute * 10,
			},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 1.
		require.Equal(t, uint(1), reg.OpenRound())
	})
	t.Run("in distant epoch", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(-100*time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute,
			},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 100.
		require.Equal(t, uint(100), reg.OpenRound())
	})
}

// Test if Proof of Work challenge is rotated every round.
// The challenge should be changed to the root of PoET proof Merkle tree
// of the previous round.
func TestPowChallengeRotation(t *testing.T) {
	genesis := time.Now()

	proofs := make(chan shared.NIP, 1)

	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(proofs)
	workerSvc.EXPECT().
		ExecuteRound(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, epoch uint, _ []byte) error {
			select {
			case proofs <- shared.NIP{
				MerkleProof: shared.MerkleProof{
					Root: []byte{1, 2, 3, 4},
				},
				Epoch: epoch,
			}:
			default:
			}
			return nil
		}).
		AnyTimes()

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&server.RoundConfig{EpochDuration: 10 * time.Millisecond},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	params0 := r.PowParams()
	require.NotEqual(t, []byte{1, 2, 3, 4}, params0.Challenge)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	require.Eventually(t, func() bool {
		return !bytes.Equal([]byte{1, 2, 3, 4}, r.PowParams().Challenge)
	}, time.Second, time.Millisecond)

	cancel()
	require.NoError(t, eg.Wait())
}

func TestRecoveringRoundInProgress(t *testing.T) {
	req := require.New(t)
	genesis := time.Now()

	roundCfg := server.RoundConfig{
		EpochDuration: time.Hour,
		PhaseShift:    time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	verifier := mocks.NewMockPowVerifier(gomock.NewController(t))
	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))
	workerSvc.EXPECT().ExecuteRound(gomock.Any(), gomock.Eq(uint(0)), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint, _ []byte) error {
			cancel()
			return nil
		},
	)

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&roundCfg,
		registration.WithPowVerifier(verifier),
	)
	req.NoError(err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	req.NoError(eg.Wait())

	// Restart the registration service.
	// The round in progress should be recovered and executed again.
	ctx, cancel = context.WithCancel(context.Background())
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))
	workerSvc.EXPECT().ExecuteRound(gomock.Any(), gomock.Eq(uint(0)), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint, _ []byte) error {
			cancel()
			return nil
		},
	)
	req.NoError(r.Run(ctx))
}
