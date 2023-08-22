package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/migrations"
)

func TestMigrate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PoetDir = t.TempDir()
	cfg.DataDir = t.TempDir()
	cfg.DbDir = t.TempDir()
	require.NoError(t, migrations.Migrate(context.Background(), cfg))
}
