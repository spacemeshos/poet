package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadingNonExistingConfigFile(t *testing.T) {
	cfg := Config{
		ConfigFile: "non-existing-file",
	}
	_, err := ReadConfigFile(&cfg)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestReadConfigFile(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	cfg := &Config{
		ConfigFile: filepath.Join(dir, "config.ini"),
	}
	err := os.WriteFile(cfg.ConfigFile, []byte("datadir = /tmp"), 0o600)
	require.NoError(t, err)

	cfg, err = ReadConfigFile(cfg)
	require.NoError(t, err)
	require.Equal(t, "/tmp", cfg.DataDir)
}

func TestReadConfigFilePathNotSet(t *testing.T) {
	cfg, err := ReadConfigFile(&Config{})
	require.NoError(t, err)
	require.Equal(t, &Config{}, cfg)
}
