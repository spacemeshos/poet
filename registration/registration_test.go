package registration_test

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/registration/mocks"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/transport"
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

	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&roundCfg,
	)
	req.NoError(err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	// Submit challenge
	epoch, _, err := r.Submit(
		context.Background(),
		challenge,
		nodeID,
		nil,
		time.Time{},
	)
	req.NoError(err)
	req.Equal(uint(0), epoch)

	// Try again - it should return the same result
	epoch, _, err = r.Submit(context.Background(), challenge, nodeID, nil, time.Time{})
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

func TestWorkingWithoutWorkerService(t *testing.T) {
	t.Parallel()

	reg, err := registration.New(
		context.Background(),
		time.Now(),
		t.TempDir(),
		transport.NewInMemory(),
		&server.RoundConfig{EpochDuration: time.Millisecond * 10},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reg.Close()) })

	var eg errgroup.Group
	ctx, cancel := context.WithCancel(logging.NewContext(context.Background(), zaptest.NewLogger(t)))
	defer cancel()
	eg.Go(func() error { return reg.Run(ctx) })

	// Verify that registration keeps opening rounds even without the worker service.
	// Only check if the round number is incremented to not rely on the exact timing.
	for i := 0; i < 3; i++ {
		round := reg.OpenRound()
		require.Eventually(t, func() bool { return reg.OpenRound() > round }, time.Second, time.Millisecond*10)
	}
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

func Test_GetCertifierInfo(t *testing.T) {
	certifier := &registration.CertifierConfig{
		PubKey: registration.Base64Enc("pubkey"),
		URL:    "http://the-certifier.org",
	}

	r, err := registration.New(
		context.Background(),
		time.Now(),
		t.TempDir(),
		nil,
		server.DefaultRoundConfig(),
		registration.WithConfig(registration.Config{
			MaxRoundMembers: 10,
			Certifier:       certifier,
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })
	require.Equal(t, r.CertifierInfo(), certifier)
}

func Test_CheckCertificate(t *testing.T) {
	challenge := []byte("challenge")
	nodeID := []byte("nodeID00nodeID00nodeID00nodeID00")

	t.Run("certification check disabled (default config)", func(t *testing.T) {
		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		// missing certificate - accept
		_, _, err = r.Submit(context.Background(), challenge, nodeID, nil, time.Time{})
		require.NoError(t, err)

		// passed certificate - accept
		_, _, err = r.Submit(
			context.Background(),
			challenge,
			nodeID,
			[]byte{1, 2, 3, 4},
			time.Time{},
		)
		require.NoError(t, err)
	})
	t.Run("certification check enabled", func(t *testing.T) {
		pub, private, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)

		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
			registration.WithConfig(registration.Config{
				MaxRoundMembers: 10,
				Certifier: &registration.CertifierConfig{
					PubKey: registration.Base64Enc(pub),
				},
			}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		// missing certificate - reject
		_, _, err = r.Submit(context.Background(), challenge, nodeID, nil, time.Time{})
		require.Error(t, err)

		// valid certificate
		signature := ed25519.Sign(private, nodeID)
		_, _, err = r.Submit(context.Background(), challenge, nodeID, signature, time.Time{})
		require.NoError(t, err)

		// invalid certificate
		_, _, err = r.Submit(context.Background(), challenge, nodeID, []byte{1, 2, 3, 4}, time.Time{})
		require.ErrorIs(t, err, registration.ErrInvalidCertificate)
	})
}
