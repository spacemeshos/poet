package migrations

import (
	"context"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/logging"
)

func Migrate(ctx context.Context, cfg *config.Config) error {
	ctx = logging.NewContext(ctx, logging.FromContext(ctx).Named("migrations"))
	if err := migrateDbDir(ctx, cfg); err != nil {
		return err
	}
	return nil
}
