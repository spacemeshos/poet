package signing

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/spacemeshos/go-scale"
)

var (
	ErrSigningFailed    = errors.New("couldn't sign")
	ErrSignatureInvalid = errors.New("signature is invalid")
	ErrInvalidPubkeyLen = errors.New("pubkey has invalid length")
)

// Signed represents a signed T data.
// It provides a read-only access to it.
type Signed[T any] interface {
	// Data retrieves the underlying data.
	// The received data is READ ONLY.
	Data() *T
	PubKey() []byte
	Signature() []byte
}

// signedData is a holder of data T which is
// guaranteed to be signed. It implements Signed[T] interface.
type signedData[T any] struct {
	data      T
	pubkey    []byte
	signature []byte
}

func (d *signedData[T]) Data() *T {
	return &d.data
}

func (d *signedData[T]) PubKey() []byte {
	return d.pubkey
}

func (d *signedData[T]) Signature() []byte {
	return d.signature
}

type notHashed struct{}

func (notHashed) HashFunc() crypto.Hash { return crypto.Hash(0) }

type encodable[P any] interface {
	scale.Encodable
	*P
}

// Sign signs data with given signer.
// *T must implement scale.Encodable which is constrained by Encodable.
func Sign[T any, Encodable encodable[T]](data T, signer crypto.Signer, pubkey []byte) (Signed[T], error) {
	var dataBuf bytes.Buffer
	if _, err := Encodable(&data).EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, fmt.Errorf("failed to serialize data (%w)", err)
	}
	signature, err := signer.Sign(nil, dataBuf.Bytes(), notHashed{})
	if err != nil {
		return nil, fmt.Errorf("%w (%v)", ErrSigningFailed, err)
	}
	return &signedData[T]{
		data:      data,
		pubkey:    pubkey,
		signature: signature,
	}, nil
}

// NewFromScaleEncodable constructs Signed[T] from a T.
// *T must implement scale.Encodable which is constrained by Encodable.
func NewFromScaleEncodable[T any, Encodable encodable[T]](data T, signature, pubkey []byte) (Signed[T], error) {
	// Serialize it for signature verification
	var dataBuf bytes.Buffer
	if _, err := Encodable(&data).EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, err
	}
	if l := len(pubkey); l != ed25519.PublicKeySize {
		return nil, ErrInvalidPubkeyLen
	}
	if !ed25519.Verify(pubkey, dataBuf.Bytes(), signature) {
		return nil, ErrSignatureInvalid
	}

	return &signedData[T]{
		data:      data,
		pubkey:    pubkey,
		signature: signature,
	}, nil
}
