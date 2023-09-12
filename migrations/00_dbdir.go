package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"

	"github.com/spacemeshos/poet/db"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/server"
)

func migrateDbDir(ctx context.Context, cfg *server.Config) error {
	if err := migrateProofsDb(ctx, cfg); err != nil {
		return fmt.Errorf("migrating proofs DB: %w", err)
	}
	if err := migrateRoundsDbs(ctx, cfg); err != nil {
		return fmt.Errorf("migrating rounds DBs: %w", err)
	}

	return nil
}

func migrateRoundsDbs(ctx context.Context, cfg *server.Config) error {
	roundsDataDir := filepath.Join(cfg.DataDir, "rounds")
	// check if dir exists
	if _, err := os.Stat(roundsDataDir); os.IsNotExist(err) {
		return nil
	}

	logger := logging.FromContext(ctx)
	logger.Info("migrating rounds DBs", zap.String("datadir", cfg.DataDir))
	entries, err := os.ReadDir(roundsDataDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if _, err := strconv.ParseUint(entry.Name(), 10, 32); err != nil {
			return fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}

		dbdir := filepath.Join(cfg.DbDir, "rounds", entry.Name())
		oldDbDir := filepath.Join(roundsDataDir, entry.Name(), "challengesDb")
		if err := db.Migrate(ctx, dbdir, oldDbDir); err != nil {
			return fmt.Errorf("migrating round DB from %s: %w", oldDbDir, err)
		}
	}
	return nil
}

func migrateProofsDb(ctx context.Context, cfg *server.Config) error {
	proofsDbPath := filepath.Join(cfg.DbDir, "proofs")
	oldProofsDbPath := filepath.Join(cfg.DataDir, "proofs")
	if err := db.Migrate(ctx, proofsDbPath, oldProofsDbPath); err != nil {
		return fmt.Errorf("migrating proofs DB %s -> %s: %w", oldProofsDbPath, proofsDbPath, err)
	}
	return nil
}
