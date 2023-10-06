package transport

import (
	"context"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/shared"
)

type transport interface {
	service.RegistrationService
	registration.WorkerService
}

// inMemory is an in-memory implementation of transport.
// It allows binding a Registration service with a Worker service
// in a standalone mode by an in-memory channel.
type inMemory struct {
	closedRounds chan service.ClosedRound
	proofs       chan shared.NIP
}

func NewInMemory() transport {
	return &inMemory{
		closedRounds: make(chan service.ClosedRound, 1),
		proofs:       make(chan shared.NIP, 1),
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
	default:
		logging.FromContext(ctx).Info("nobody listens to closed rounds - dropping")
		return nil
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
