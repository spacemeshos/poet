package registration

import (
	"time"

	"go.uber.org/zap/zapcore"
)

func DefaultConfig() Config {
	return Config{
		MaxRoundMembers:     1 << 32,
		SubmitFlushInterval: 100 * time.Millisecond,
		MaxSubmitBatchSize:  1000,
	}
}

type Config struct {
	// FIXME: remove depreacated PoW
	PowDifficulty uint `long:"pow-difficulty" description:"(DEPRECATED) PoW difficulty (in the number of leading zero bits)"`

	MaxRoundMembers     int           `long:"max-round-members"     description:"the maximum number of members in a round"`
	MaxSubmitBatchSize  int           `long:"max-submit-batch-size" description:"The maximum number of challenges to submit in a single batch"`
	SubmitFlushInterval time.Duration `long:"submit-flush-interval" description:"The interval between flushes of the submit queue"`

	Certifier *CertifierConfig
}

type CertifierConfig struct {
	URL    string `long:"certifier-url"    description:"The URL of the certifier service"`
	PubKey []byte `long:"certifier-pubkey" description:"The public key of the certifier service"`
}

// implement zap.ObjectMarshaler interface.
func (c CertifierConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("url", c.URL)
	enc.AddBinary("pubkey", c.PubKey)
	return nil
}
