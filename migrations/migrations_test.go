package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/migrations"
	"github.com/spacemeshos/poet/server"
)

func TestMigrate(t *testing.T) {
	cfg := server.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.DataDir = t.TempDir()
	cfg.DbDir = t.TempDir()
	require.NoError(t, migrations.Migrate(context.Background(), cfg))
}
