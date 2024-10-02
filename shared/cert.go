package shared

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/spacemeshos/go-scale"
)

//go:generate scalegen -types CertOnWire,UnixTimestamp

var (
	ErrCertExpired           = errors.New("certificate expired")
	ErrCertSignatureMismatch = errors.New("signature mismatch")
	ErrCertDataMismatch      = errors.New("pubkey mismatch")
)

type UnixTimestamp struct {
	Inner uint64
}

func (u *UnixTimestamp) Time() *time.Time {
	if u == nil {
		return nil
	}
	exp := time.Unix(int64(u.Inner), 0).UTC()
	return &exp
}

// CertOnWire is a certificate that is sent `over the wire`.
// This type is encoded with scale for signing.
type CertOnWire struct {
	Pubkey     []byte `scale:"max=32"`
	Expiration *UnixTimestamp
}

// OpaqueCert is a certificate that is sent `over the wire`.
// It holds the encoded certificate and its signature.
type OpaqueCert struct {
	Data      []byte // scale-encoded CertOnWire
	Signature []byte // signature of Data
}

func (c *OpaqueCert) Decode() (*Cert, error) {
	return DecodeCert(c.Data)
}

const CertKeyHintSize = 4

type CertKeyHint [CertKeyHintSize]byte

type Cert struct {
	// The ID that this certificate allows registration for.
	Pubkey []byte `scale:"max=32"`
	// The expiration time of the certificate.
	// The certificate doesn't expire if this field is nil.
	Expiration *time.Time
}

func DecodeCert(d []byte) (*Cert, error) {
	var c CertOnWire
	if _, err := c.DecodeScale(scale.NewDecoder(bytes.NewBuffer(d))); err != nil {
		return nil, err
	}

	return &Cert{
		Pubkey:     c.Pubkey,
		Expiration: c.Expiration.Time(),
	}, nil
}

func EncodeCert(c *Cert) ([]byte, error) {
	certOnWire := CertOnWire{
		Pubkey: c.Pubkey,
	}
	if c.Expiration != nil {
		certOnWire.Expiration = new(UnixTimestamp)
		*certOnWire.Expiration = UnixTimestamp{uint64(c.Expiration.Unix())}
	}
	var buf bytes.Buffer
	if _, err := certOnWire.EncodeScale(scale.NewEncoder(&buf)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func VerifyCertificate(certificate *OpaqueCert, certifierPubKey, nodeID []byte) (*Cert, error) {
	if !ed25519.Verify(certifierPubKey, certificate.Data, certificate.Signature) {
		return nil, ErrCertSignatureMismatch
	}
	decoded, err := certificate.Decode()
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}
	if !bytes.Equal(decoded.Pubkey, nodeID) {
		return nil, ErrCertDataMismatch
	}
	if decoded.Expiration != nil && decoded.Expiration.Before(time.Now()) {
		return nil, fmt.Errorf("%w at %v", ErrCertExpired, decoded.Expiration)
	}
	return decoded, nil
}
