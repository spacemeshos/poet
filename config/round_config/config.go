package round_config

import (
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	defaultEpochDuration = 30 * time.Second
	defaultPhaseShift    = 15 * time.Second
	defaultCycleGap      = 10 * time.Second
)

type Config struct {
	EpochDuration time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift    time.Duration `long:"phase-shift"`
	CycleGap      time.Duration `long:"cycle-gap"`
}

func DefaultConfig() Config {
	return Config{
		EpochDuration: defaultEpochDuration,
		PhaseShift:    defaultPhaseShift,
		CycleGap:      defaultCycleGap,
	}
}

func (c *Config) RoundStart(genesis time.Time, epoch uint) time.Time {
	return genesis.Add(c.PhaseShift).Add(c.EpochDuration * time.Duration(epoch))
}

func (c *Config) RoundEnd(genesis time.Time, epoch uint) time.Time {
	return c.RoundStart(genesis, epoch).Add(c.EpochDuration).Add(-c.CycleGap)
}

func (c *Config) RoundDuration() time.Duration {
	return c.EpochDuration - c.CycleGap
}

// Calculate ID of the open round at a given point in time.
func (c *Config) OpenRoundId(genesis, when time.Time) uint {
	sinceGenesis := when.Sub(genesis)
	if sinceGenesis < c.PhaseShift {
		return 0
	}
	return uint(int(sinceGenesis-c.PhaseShift)/int(c.EpochDuration)) + 1
}

// implement zap.ObjectMarshaler interface.
func (c Config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddDuration("epoch-duration", c.EpochDuration)
	enc.AddDuration("phase-shift", c.PhaseShift)
	enc.AddDuration("cycle-gap", c.CycleGap)

	return nil
}
