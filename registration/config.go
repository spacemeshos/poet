package registration

import (
	"encoding/base64"
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

//nolint:lll
type Config struct {
	// FIXME: remove deprecated PoW
	PowDifficulty uint `long:"pow-difficulty" description:"(DEPRECATED) PoW difficulty (in the number of leading zero bits)"`

	MaxRoundMembers     int           `long:"max-round-members"     description:"the maximum number of members in a round"`
	MaxSubmitBatchSize  int           `long:"max-submit-batch-size" description:"The maximum number of challenges to submit in a single batch"`
	SubmitFlushInterval time.Duration `long:"submit-flush-interval" description:"The interval between flushes of the submit queue"`

	Certifier *CertifierConfig
}

type Base64Enc []byte

func (k *Base64Enc) UnmarshalFlag(value string) error {
	b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	*k = b
	return nil
}

func (k *Base64Enc) Bytes() []byte {
	return *k
}

// Configuration of registration certificates.
//
// Support for registering with certificates is enabled by setting at least one of:
//   - `PubKey`
//   - `TrustedKeysDirPath`
//
// The `PubKey` is the public key of the certifier exposed via the URL field.
// It is legacy and doesn't require the client to pass a key hint. A Submit() without
// a hint will use this key.
//
// The `TrustedKeysDirPath` allows for configuring more than one key, but without
// a URL to the related certifier(s). To use certificates signed with the private part of
// these keys, the client must pass a key hint to Submit().
// The keys are loaded on boot and on demand and together with the `PubKey` form a set
// of trusted keys.
//
//nolint:lll
type CertifierConfig struct {
	URL                string    `long:"certifier-url"             description:"The URL of the certifier service"`
	PubKey             Base64Enc `long:"certifier-pubkey"          description:"The public key of the certifier service (base64 encoded)"`
	TrustedKeysDirPath string    `long:"trusted-pub-keys-dir-path" description:"The path to directory with trusted public keys"`
}

// implement zap.ObjectMarshaler interface.
func (c *CertifierConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("url", c.URL)
	enc.AddString("pubkey", base64.StdEncoding.EncodeToString(c.PubKey))
	return nil
}
