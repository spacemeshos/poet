package signing

import (
	"errors"

	"github.com/spacemeshos/ed25519"
)

var ErrSignatureInvalid = errors.New("signature is invalid")

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

func NewFromBytes(data []byte, signature []byte) (Signed[[]byte], error) {
	if len(signature) != ed25519.SignatureSize {
		return nil, ErrSignatureInvalid
	}

	// Verify the signature
	pubkey, err := ed25519.ExtractPublicKey(data, signature)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify2(pubkey, data, signature) {
		return nil, ErrSignatureInvalid
	}

	return &signedData[[]byte]{
		data:      data,
		pubkey:    pubkey,
		signature: signature,
	}, nil
}
