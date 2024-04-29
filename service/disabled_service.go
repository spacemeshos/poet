package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
)

type disabledService struct {
	reg RegistrationService
}

func NewDisabledService(reg RegistrationService) *disabledService {
	return &disabledService{
		reg: reg,
	}
}

func (s *disabledService) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx).Named("dummy-worker")
	closedRounds := s.reg.RegisterForRoundClosed(ctx)

	for {
		select {
		case closedRound := <-closedRounds:
			logger := logger.With(zap.Uint("epoch", closedRound.Epoch))
			logger.Info("received round to execute")
			if err := s.reg.NewProof(ctx, shared.NIP{Epoch: closedRound.Epoch}); err != nil {
				logger.Error("failed to publish new proof", zap.Error(err))
			}
			logger.Info("published empty proof")
		case <-ctx.Done():
			logger.Info("service shutting down")
			return nil
		}
	}
}
