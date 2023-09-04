package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/spacemeshos/poet/db"
)

var kvs = map[string][]byte{
	"key":  []byte("value"),
	"key2": []byte("value2"),
	"key3": []byte("value3"),
}

func TestMigrateDb(t *testing.T) {
	// open a database and write some data
	oldDbPath := t.TempDir()
	oldDb, err := leveldb.OpenFile(oldDbPath, nil)
	require.NoError(t, err)
	defer oldDb.Close()
	for k, v := range kvs {
		require.NoError(t, oldDb.Put([]byte(k), v, nil))
	}
	oldDb.Close()

	// migrate the database
	newDbPath := t.TempDir()
	require.NoError(t, db.Migrate(context.Background(), newDbPath, oldDbPath))

	// open the new database and check that the data was copied
	newDb, err := leveldb.OpenFile(newDbPath, nil)
	require.NoError(t, err)
	defer newDb.Close()

	for k, v := range kvs {
		value, err := newDb.Get([]byte(k), nil)
		require.NoError(t, err)
		require.Equal(t, v, value)
	}

	// old DB should be removed
	_, err = os.Stat(oldDbPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSkipMigrateInPlace(t *testing.T) {
	// open a database and write some data
	dbPath := t.TempDir()
	database, err := leveldb.OpenFile(dbPath, nil)
	require.NoError(t, err)
	defer database.Close()
	for k, v := range kvs {
		require.NoError(t, database.Put([]byte(k), v, nil))
	}
	database.Close()

	// migrate the database
	require.NoError(t, db.Migrate(context.Background(), dbPath, dbPath))

	// open the new database and check that the data was copied
	database, err = leveldb.OpenFile(dbPath, nil)
	require.NoError(t, err)
	defer database.Close()

	for k, v := range kvs {
		value, err := database.Get([]byte(k), nil)
		require.NoError(t, err)
		require.Equal(t, v, value)
	}
}

func TestSkipMigrateSrcDoesntExist(t *testing.T) {
	require.NoError(t, db.Migrate(context.Background(), t.TempDir(), "i-dont-exist"))
}