package registration

import "time"

func DefaultConfig() Config {
	return Config{
		MaxRoundMembers:     1 << 32,
		SubmitFlushInterval: 100 * time.Millisecond,
		MaxSubmitBatchSize:  1000,
	}
}

type Config struct {
	PowDifficulty uint `long:"pow-difficulty" description:"PoW difficulty (in the number of leading zero bits)"`

	MaxRoundMembers     int           `long:"max-round-members"     description:"the maximum number of members in a round"`
	MaxSubmitBatchSize  int           `long:"max-submit-batch-size" description:"The maximum number of challenges to submit in a single batch"`
	SubmitFlushInterval time.Duration `long:"submit-flush-interval" description:"The interval between flushes of the submit queue"`
}
