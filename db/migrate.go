package db

import (
	"context"
	"fmt"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
)

// Temporary code to migrate a database to a new location.
// It opens both DBs and copies all the data from the old DB to the new one.
func Migrate(ctx context.Context, targetDbDir, oldDbDir string) error {
	log := logging.FromContext(ctx)
	log.Info(
		"attempting DB location migration",
		zap.String("oldDbDir", oldDbDir),
		zap.String("targetDbDir", targetDbDir),
	)
	if oldDbDir == targetDbDir {
		log.Debug("skipping in-place DB migration")
		return nil
	}

	oldDb, err := leveldb.OpenFile(oldDbDir, &opt.Options{ErrorIfMissing: true})
	switch {
	case os.IsNotExist(err):
		log.Debug("skipping DB migration - old DB doesn't exist")
		return nil
	case err != nil:
		return fmt.Errorf("opening old DB: %w", err)
	}
	defer oldDb.Close()

	targetDb, err := leveldb.OpenFile(targetDbDir, &opt.Options{ErrorIfExist: true})
	if err != nil {
		return fmt.Errorf("opening target DB: %w", err)
	}
	defer targetDb.Close()

	tx, err := targetDb.OpenTransaction()
	if err != nil {
		return fmt.Errorf("opening new DB transaction: %w", err)
	}
	iter := oldDb.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		if err := tx.Put(iter.Key(), iter.Value(), nil); err != nil {
			tx.Discard()
			return fmt.Errorf("migrating key %X: %w", iter.Key(), err)
		}
	}
	iter.Release()
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing DB transaction: %w", err)
	}

	// Remove old DB
	log.Info("removing the old DB")
	if err := oldDb.Close(); err != nil {
		return fmt.Errorf("closing old DB: %w", err)
	}
	if err := os.RemoveAll(oldDbDir); err != nil {
		return fmt.Errorf("removing old DB: %w", err)
	}
	log.Info("DB migrated to new location")
	return nil
}
