package service_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/service/mocks"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/transport"
	"github.com/spacemeshos/poet/verifier"
)

func TestService_Recovery(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()
	genesis := time.Now().Add(time.Second)

	registration := mocks.NewMockRegistrationService(gomock.NewController(t))
	proofs := make(chan shared.NIP, 1)
	closedRoundsChan := make(chan service.ClosedRound)

	// Create a new service instance.
	registration.EXPECT().RegisterForRoundClosed(gomock.Any()).Return(closedRoundsChan)
	s, err := service.New(
		context.Background(),
		genesis,
		tempdir,
		registration,
		&server.RoundConfig{EpochDuration: time.Hour},
	)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	// Start round 0
	membershipRoot := []byte{1, 2, 3, 4}
	closedRoundsChan <- service.ClosedRound{Epoch: 0, MembershipRoot: membershipRoot}

	cancel()
	req.NoError(eg.Wait())

	// Create a new service instance.
	registration.EXPECT().RegisterForRoundClosed(gomock.Any()).Return(closedRoundsChan)
	registration.EXPECT().
		NewProof(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, proof shared.NIP) error {
			proofs <- proof
			return nil
		})
	s, err = service.New(
		context.Background(),
		genesis,
		tempdir,
		registration,
		&server.RoundConfig{EpochDuration: time.Second},
	)
	req.NoError(err)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	eg = errgroup.Group{}
	eg.Go(func() error { return s.Run(ctx) })

	proof := <-proofs
	req.Equal(uint(0), proof.Epoch)
	err = verifier.Validate(
		proof.MerkleProof,
		hash.GenLabelHashFunc(membershipRoot),
		hash.GenMerkleHashFunc(membershipRoot),
		proof.Leaves,
		shared.T,
	)
	req.NoError(err)

	cancel()
	req.NoError(eg.Wait())
}

func TestNewService(t *testing.T) {
	req := require.New(t)
	genesis := time.Now().Add(time.Second)

	registration := mocks.NewMockRegistrationService(gomock.NewController(t))
	proofs := make(chan shared.NIP, 1)
	closedRoundsChan := make(chan service.ClosedRound)

	// Create a new service instance.
	registration.EXPECT().RegisterForRoundClosed(gomock.Any()).Return(closedRoundsChan)
	s, err := service.New(
		context.Background(),
		genesis,
		t.TempDir(),
		registration,
		&server.RoundConfig{EpochDuration: time.Second * 2},
	)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	// Run 2 rounds and verify the proofs.
	registration.EXPECT().
		NewProof(gomock.Any(), gomock.Any()).
		Times(2).
		DoAndReturn(func(_ context.Context, proof shared.NIP) error {
			proofs <- proof
			return nil
		})

	for round := 0; round < 2; round++ {
		membershipRoot := bytes.Repeat([]byte{byte(round)}, 32)
		closedRoundsChan <- service.ClosedRound{Epoch: uint(round), MembershipRoot: membershipRoot}

		proof := <-proofs
		req.Equal(uint(round), proof.Epoch)
		err = verifier.Validate(
			proof.MerkleProof,
			hash.GenLabelHashFunc(membershipRoot),
			hash.GenMerkleHashFunc(membershipRoot),
			proof.Leaves,
			shared.T,
		)
		req.NoError(err)
	}

	cancel()
	req.NoError(eg.Wait())
}

func TestSkipPastRounds(t *testing.T) {
	req := require.New(t)

	transport := transport.NewInMemory()
	s, err := service.New(
		context.Background(),
		time.Now().Add(-time.Second),
		t.TempDir(),
		transport,
		&server.RoundConfig{EpochDuration: time.Millisecond},
	)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	proofs := transport.RegisterForProofs(context.Background())
	// Try executing round in the past, should be ignored
	transport.ExecuteRound(ctx, 0, nil)

	select {
	case <-proofs:
		t.Fatal("should not receive proof")
	case <-time.After(time.Millisecond * 100):
	}

	cancel()
	req.NoError(eg.Wait())
}

func TestRecoverFinishedRound(t *testing.T) {
	req := require.New(t)

	datadir := t.TempDir()
	// manually create a round and execute it
	round, err := service.NewRound(
		filepath.Join(datadir, "rounds"),
		9876,
		service.WithMembershipRoot([]byte{1, 2, 3, 4}),
	)
	req.NoError(err)
	err = round.Execute(context.Background(), time.Now().Add(time.Millisecond*10), 0, 0)
	req.NoError(err)
	req.True(round.IsFinished())

	transport := transport.NewInMemory()
	s, err := service.New(
		context.Background(),
		time.Now(),
		datadir,
		transport,
		&server.RoundConfig{EpochDuration: time.Hour},
	)
	req.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	proofs := transport.RegisterForProofs(context.Background())
	var proof shared.NIP
	req.Eventually(func() bool {
		select {
		case p := <-proofs:
			proof = p
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)

	req.Equal(uint(9876), proof.Epoch)
	req.Eventually(func() bool {
		_, err := os.Lstat(filepath.Join(datadir, "rounds", "9876"))
		return errors.Is(err, os.ErrNotExist)
	}, time.Second*5, 10*time.Millisecond)

	cancel()
	req.NoError(eg.Wait())
}

func TestRemoveRecoveredOpenRound(t *testing.T) {
	req := require.New(t)
	ctx := logging.NewContext(context.Background(), zaptest.NewLogger(t))
	datadir := t.TempDir()

	// manually create a round and execute it
	round, err := service.NewRound(
		filepath.Join(datadir, "rounds"),
		1,
		service.WithMembershipRoot([]byte{1, 2, 3, 4}),
	)
	req.NoError(err)
	t.Cleanup(func() { assert.NoError(t, round.Teardown(ctx, false)) })
	ctxE, cancel := context.WithTimeout(ctx, time.Millisecond*10)
	defer cancel()
	err = round.Execute(ctxE, time.Now().Add(time.Hour), 0, 0)
	req.ErrorIs(err, context.DeadlineExceeded)

	// manually create an open round
	openRound, err := service.NewRound(
		filepath.Join(datadir, "rounds"),
		2,
		service.WithMembershipRoot([]byte{1, 2, 3, 4}),
	)
	req.NoError(err)
	t.Cleanup(func() { assert.NoError(t, openRound.Teardown(ctx, true)) })

	transport := transport.NewInMemory()
	s, err := service.New(
		ctx,
		time.Now(),
		datadir,
		transport,
		&server.RoundConfig{EpochDuration: time.Hour},
	)
	req.NoError(err)

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(ctx) })

	req.Eventually(func() bool {
		_, err := os.Lstat(filepath.Join(datadir, "rounds", "2"))
		return errors.Is(err, os.ErrNotExist)
	}, time.Second*5, 10*time.Millisecond)

	cancel()
	req.NoError(eg.Wait())
}

// Test that the service doesn't restart a round afresh if it's recovered execution is canceled.
// This is a regression test for https://github.com/spacemeshos/poet/issues/400
func TestServiceCancelRecoveredRound(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()
	genesis := time.Now()
	ctx := logging.NewContext(context.Background(), zaptest.NewLogger(t))

	closedRoundsChan := make(chan service.ClosedRound)
	registration := mocks.NewMockRegistrationService(gomock.NewController(t))
	registration.EXPECT().RegisterForRoundClosed(gomock.Any()).Return(closedRoundsChan)

	// Create a new service instance.
	s, err := service.New(
		ctx,
		genesis,
		tempdir,
		registration,
		&server.RoundConfig{EpochDuration: time.Hour},
	)
	req.NoError(err)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return s.Run(runCtx) })

	// Start round 0
	membershipRoot := []byte{1, 2, 3, 4}
	closedRoundsChan <- service.ClosedRound{Epoch: 0, MembershipRoot: membershipRoot}

	cancel()
	req.NoError(eg.Wait())

	// Create a new service instance.
	// Should not register for closed rounds as the recovered round will be canceled.
	s, err = service.New(
		ctx,
		genesis,
		tempdir,
		registration,
		&server.RoundConfig{EpochDuration: time.Hour},
	)
	req.NoError(err)

	runCtx, cancel = context.WithCancel(ctx)
	cancel()
	req.NoError(s.Run(runCtx))
}
