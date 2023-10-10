package service

import (
	"context"
	"time"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
	"go.uber.org/zap"
)

type disabledService struct {
	reg      RegistrationService
	roundCfg roundConfig
	genesis  time.Time
}

func NewDisabledService(reg RegistrationService, genesis time.Time, roundCfg roundConfig) *disabledService {
	return &disabledService{
		reg:      reg,
		roundCfg: roundCfg,
		genesis:  genesis,
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
			end := s.roundCfg.RoundEnd(s.genesis, closedRound.Epoch)
			if end.Before(time.Now()) {
				logger.Info("skipping past round", zap.Time("expected end", end))
				continue
			}

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
