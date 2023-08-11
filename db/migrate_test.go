package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/spacemeshos/poet/db"
)

func TestMigrateDb(t *testing.T) {
	// open a database and write some data
	oldDbPath := t.TempDir()
	oldDb, err := leveldb.OpenFile(oldDbPath, nil)
	require.NoError(t, err)
	defer oldDb.Close()
	require.NoError(t, oldDb.Put([]byte("key"), []byte("value"), nil))
	require.NoError(t, oldDb.Put([]byte("key2"), []byte("value2"), nil))
	oldDb.Close()

	// migrate the database
	newDbPath := t.TempDir()
	require.NoError(t, db.Migrate(context.Background(), newDbPath, oldDbPath))

	// open the new database and check that the data was copied
	newDb, err := leveldb.OpenFile(newDbPath, nil)
	require.NoError(t, err)
	defer newDb.Close()
	value, err := newDb.Get([]byte("key"), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), value)
	value, err = newDb.Get([]byte("key2"), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("value2"), value)

	// old DB should be removed
	_, err = os.Stat(oldDbPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}
