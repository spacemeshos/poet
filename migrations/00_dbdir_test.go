package migrations

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/spacemeshos/poet/server"
)

func TestMigrateRoundsDb(t *testing.T) {
	// Prepare
	cfg := server.Config{
		DataDir: t.TempDir(),
		DbDir:   t.TempDir(),
	}
	for i := 0; i < 5; i++ {
		id := strconv.Itoa(i)
		oldDb, err := leveldb.OpenFile(filepath.Join(cfg.DataDir, "rounds", id, "challengesDb"), nil)
		require.NoError(t, err)
		defer oldDb.Close()
		require.NoError(t, oldDb.Put([]byte(id+"key"), []byte("value"), nil))
		require.NoError(t, oldDb.Put([]byte(id+"key2"), []byte("value2"), nil))
		oldDb.Close()
	}
	// Act
	require.NoError(t, migrateRoundsDbs(context.Background(), &cfg))

	// Verify
	for i := 0; i < 5; i++ {
		id := strconv.Itoa(i)
		db, err := leveldb.OpenFile(filepath.Join(cfg.DbDir, "rounds", id), nil)
		require.NoError(t, err)
		defer db.Close()

		v, err := db.Get([]byte(id+"key"), nil)
		require.NoError(t, err)
		require.Equal(t, []byte("value"), v)

		v, err = db.Get([]byte(id+"key2"), nil)
		require.NoError(t, err)
		require.Equal(t, []byte("value2"), v)

		db.Close()
	}
}

func TestMigrateRoundsDb_NothingToMigrate(t *testing.T) {
	cfg := server.Config{
		DataDir: t.TempDir(),
		DbDir:   t.TempDir(),
	}
	require.NoError(t, migrateRoundsDbs(context.Background(), &cfg))
}
