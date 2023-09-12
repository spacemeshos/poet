package migrations

import (
	"context"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/server"
)

func Migrate(ctx context.Context, cfg *server.Config) error {
	ctx = logging.NewContext(ctx, logging.FromContext(ctx).Named("migrations"))
	if err := migrateDbDir(ctx, cfg); err != nil {
		return err
	}
	return nil
}
