package signing

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spacemeshos/ed25519"
	"github.com/spacemeshos/go-scale"
)

var (
	ErrSigningFailed    = errors.New("couldn't sign")
	ErrSignatureInvalid = errors.New("signature is invalid")
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

// Signer signs data with pubkey embedding and gives access to pubkey.
// Should implement https://pkg.go.dev/github.com/spacemeshos/ed25519#readme-sign2.
type Signer interface {
	Sign([]byte) []byte
	PublicKey() []byte
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

type encodable[P any] interface {
	scale.Encodable
	*P
}

// Sign signs data with given signer.
// *T must implement scale.Encodable which is constrained by Encodable.
func Sign[T any, Encodable encodable[T]](data T, signer Signer) (Signed[T], error) {
	var dataBuf bytes.Buffer
	if _, err := Encodable(&data).EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, fmt.Errorf("failed to serialize data (%w)", err)
	}
	return &signedData[T]{
		data:      data,
		pubkey:    signer.PublicKey(),
		signature: signer.Sign(dataBuf.Bytes()),
	}, nil
}

// NewFromScaleEncodable constructs Signed[T] from a T.
// It verifies the signature and extracts the public key.
// *T must implement scale.Encodable which is constrained by Encodable.
func NewFromScaleEncodable[T any, Encodable encodable[T]](data T, signature []byte) (Signed[T], error) {
	if len(signature) != ed25519.SignatureSize {
		return nil, ErrSignatureInvalid
	}
	// Serialize it for signature verification
	var dataBuf bytes.Buffer
	if _, err := Encodable(&data).EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, err
	}
	// Verify the signature
	pubkey, err := ed25519.ExtractPublicKey(dataBuf.Bytes(), signature)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify2(pubkey, dataBuf.Bytes(), signature) {
		return nil, ErrSignatureInvalid
	}

	return &signedData[T]{
		data:      data,
		pubkey:    pubkey,
		signature: signature,
	}, nil
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
