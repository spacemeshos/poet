package round_config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculatingOpenRoundId(t *testing.T) {
	t.Parallel()
	t.Run("before genesis", func(t *testing.T) {
		now := time.Now()
		cfg := Config{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(time.Minute), now)
		require.Equal(t, uint(0), openRoundId)
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		now := time.Now()
		cfg := Config{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute * 10,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-time.Minute), now)
		require.Equal(t, uint(0), openRoundId)
	})
	t.Run("in first epoch, after phase shift", func(t *testing.T) {
		now := time.Now()
		cfg := Config{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-2*time.Minute), now)
		require.Equal(t, uint(1), openRoundId)
	})
	t.Run("distant epoch", func(t *testing.T) {
		now := time.Now()
		cfg := Config{
			EpochDuration: time.Hour,
			PhaseShift:    time.Minute,
		}
		openRoundId := cfg.OpenRoundId(now.Add(-time.Hour*100), now)
		require.Equal(t, uint(100), openRoundId)
	})
}
