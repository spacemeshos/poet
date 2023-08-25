package registration

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/spacemeshos/poet/shared"
)

var (
	ErrInvalidPow       = errors.New("invalid proof of work")
	ErrInvalidPowParams = errors.New("invalid proof of work parameters")
)

//go:generate mockgen -package mocks -destination mocks/pow_verifier.go . PowVerifier

type PowVerifier interface {
	// Verify the proof of work.
	//
	// PoW hash is ripemd256(powChallenge || nodeID || poetChallenge || nonce)
	Verify(poetChallenge, nodeID []byte, nonce uint64) error
	Params() PowParams
	SetParams(PowParams)
}

type PowParams struct {
	Challenge []byte
	// Difficulty is the number of leading zero bits required in the hash
	Difficulty uint
}

func NewPowParams(challenge []byte, difficulty uint) PowParams {
	return PowParams{
		Challenge:  challenge,
		Difficulty: difficulty,
	}
}

func (p PowParams) Equal(other PowParams) bool {
	if p.Difficulty != other.Difficulty {
		return false
	}

	return bytes.Equal(p.Challenge, other.Challenge)
}

type powVerifier struct {
	params PowParams
}

func (v *powVerifier) Params() PowParams {
	return v.params
}

func (v *powVerifier) SetParams(p PowParams) {
	v.params = p
}

func NewPowVerifier(params PowParams) PowVerifier {
	return &powVerifier{params}
}

// Verify implements PowVerifier.
func (v *powVerifier) Verify(poetChallenge, nodeID []byte, nonce uint64) error {
	if len(nodeID) != 32 {
		return fmt.Errorf("%w: invalid nodeID length: %d", ErrInvalidPow, len(nodeID))
	}

	hash := shared.CalcSubmitPowHash(v.params.Challenge, poetChallenge, nodeID, nil, nonce)

	if !shared.CheckLeadingZeroBits(hash, v.params.Difficulty) {
		return fmt.Errorf("%w: invalid leading zero bits", ErrInvalidPow)
	}

	return nil
}

type powVerifiers struct {
	mutex    sync.RWMutex
	previous PowVerifier
	current  PowVerifier
}

func (v *powVerifiers) SetParams(p PowParams) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.previous = &powVerifier{
		params: v.current.Params(),
	}
	v.current = &powVerifier{
		params: p,
	}
}

func (v *powVerifiers) Params() PowParams {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.current.Params()
}

func (v *powVerifiers) VerifyWithParams(poetChallenge, nodeID []byte, nonce uint64, params PowParams) error {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	if v.current.Params().Equal(params) {
		return v.current.Verify(poetChallenge, nodeID, nonce)
	}

	if v.previous != nil && v.previous.Params().Equal(params) {
		return v.previous.Verify(poetChallenge, nodeID, nonce)
	}
	return ErrInvalidPowParams
}
