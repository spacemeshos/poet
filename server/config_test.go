package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestCalculatingOpenRoundId(t *testing.T) {
	t.Parallel()
	t.Run("before genesis", func(t *testing.T) {
		now := time.Now()
		cfg := RoundConfig{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(time.Minute), now)
		require.Equal(t, uint(0), openRoundId)
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		now := time.Now()
		cfg := RoundConfig{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute * 10,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-time.Minute), now)
		require.Equal(t, uint(0), openRoundId)
	})
	t.Run("in first epoch, after phase shift", func(t *testing.T) {
		now := time.Now()
		cfg := RoundConfig{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-2*time.Minute), now)
		require.Equal(t, uint(1), openRoundId)
	})
	t.Run("distant epoch", func(t *testing.T) {
		now := time.Now()
		cfg := RoundConfig{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-time.Hour*100), now)
		require.Equal(t, uint(100), openRoundId)
	})
}
