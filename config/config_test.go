package config

import (
	"io/ioutil"
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

func TestReadConfigFromProvidedDefault(t *testing.T) {
	// Arrange
	cfg := &Config{}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.ini")
	err := ioutil.WriteFile(cfgFile, []byte("datadir = /tmp"), 0o600)
	require.NoError(t, err)
	t.Run("provided default don't exist", func(t *testing.T) {
		cfg, err = ReadConfigFile(cfg, "i-dont-exist")
		require.NoError(t, err)
		require.Equal(t, cfg, &Config{})
	})
	t.Run("first default", func(t *testing.T) {
		cfg, err = ReadConfigFile(cfg, cfgFile)
		require.NoError(t, err)
		require.Equal(t, "/tmp", cfg.DataDir)
	})
	t.Run("first not existing - try second", func(t *testing.T) {
		cfg, err = ReadConfigFile(cfg, "i-dont-exist", cfgFile)
		require.NoError(t, err)
		require.Equal(t, "/tmp", cfg.DataDir)
	})
}

func TestDefaultConfigPaths(t *testing.T) {
	paths := DefaultCfgPaths()
	require.Len(t, paths, 2)
}
