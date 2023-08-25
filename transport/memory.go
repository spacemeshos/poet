package transport

import (
	"context"

	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/shared"
)

type transport interface {
	service.RegistrationService
	registration.WorkerService
}

type inMemory struct {
	closedRounds chan service.ClosedRound
	proofs       chan shared.NIP
}

func NewInMemory() transport {
	return &inMemory{
		closedRounds: make(chan service.ClosedRound, 2),
		proofs:       make(chan shared.NIP, 2),
	}
}

// Implement registration.WorkerService.
func (m *inMemory) ExecuteRound(ctx context.Context, epoch uint, membershipRoot []byte) error {
	select {
	case m.closedRounds <- service.ClosedRound{
		Epoch:          epoch,
		MembershipRoot: membershipRoot,
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *inMemory) RegisterForProofs(ctx context.Context) <-chan shared.NIP {
	return m.proofs
}

// Implement service.RegistrationService.
func (m *inMemory) RegisterForRoundClosed(ctx context.Context) <-chan service.ClosedRound {
	return m.closedRounds
}

func (m *inMemory) NewProof(ctx context.Context, proof shared.NIP) error {
	select {
	case m.proofs <- proof:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
