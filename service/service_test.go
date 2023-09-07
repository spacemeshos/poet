package service_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/config/round_config"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/service/mocks"
	"github.com/spacemeshos/poet/shared"
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
	s, err := service.NewService(
		context.Background(),
		genesis,
		tempdir,
		registration,
		service.WithRoundConfig(round_config.Config{
			EpochDuration: time.Hour,
		}),
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
	s, err = service.NewService(
		context.Background(),
		genesis,
		tempdir,
		registration,
		service.WithRoundConfig(round_config.Config{
			EpochDuration: time.Second,
		}),
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
	s, err := service.NewService(
		context.Background(),
		genesis,
		t.TempDir(),
		registration,
		service.WithRoundConfig(round_config.Config{
			EpochDuration: time.Second,
		}),
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
