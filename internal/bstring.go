package internal

import (
	"crypto/rand"
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"math/big"
	"math/bits"
	"strconv"
)

type SMBinaryStringFactory struct{}

func NewSMBinaryStringFactory() shared.BinaryStringFactory {
	return &SMBinaryStringFactory{}
}

type SMBinaryString struct {
	v uint64 // stored value
	d uint   // number of binary digits to display
	f *SMBinaryStringFactory
}

// Create a new BinaryString from a string of 0s and 1s, e.g. "00111"
// Returns an error if s is not a valid binary string, e.g. it contains chars
// other then 0 or 1
// Any leading 0s will be included in the result
func (f *SMBinaryStringFactory) NewBinaryString(s string) (shared.BinaryString, error) {

	v, err := strconv.ParseUint(s, 2, 64)
	if err != nil {
		return nil, err
	}

	res := &SMBinaryString{
		v: v,
		d: uint(len(s)),
		f: f,
	}

	return res, nil
}

// Create a new random d digits long BinaryString. e.g for digits = 4 "0110"
// d <= 63
func (f *SMBinaryStringFactory) NewRandomBinaryString(d uint) (shared.BinaryString, error) {

	if d > 63 {
		return nil, errors.New("unsupported # of digits. must be less or equals to 64")
	}

	// generate a random number with d digits

	max := uint64(math.Exp2(float64(d)))
	maxBig := new(big.Int).SetUint64(max)

	// maxNum with d digits is 2^d - 1. The following returns 0 <= nBig <= maxNum
	rndBig, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		return nil, err
	}

	v := rndBig.Uint64()
	return f.NewBinaryStringFromInt(v, d)
}


// digits must be at least as large to represent v
func (f *SMBinaryStringFactory) NewBinaryStringFromInt(v uint64, d uint) (shared.BinaryString, error) {

	l := uint(bits.Len64(v))
	if l > d {
		return nil, errors.New("invalid digits. Digits must be large enough to represent v in bits")
	}

	res := &SMBinaryString{
		v: v,
		d: d,
		f: f,
	}

	return res, nil
}

// Returns a new BinaryString with the LSB truncared. e.g. "0010" => "001"
func (s *SMBinaryString) TruncateLSB() shared.BinaryString {
	res, _ := s.f.NewBinaryStringFromInt(s.v>>1, s.d-1)
	return res
}

// Flip LSB. e.g. "0010" => "0011"
func (s *SMBinaryString) FlipLSB() shared.BinaryString {
	v := s.v ^ 1
	res, _ := s.f.NewBinaryStringFromInt(v, s.d)
	return res
}

// Get string representation. e.g. "00011"
func (s *SMBinaryString) GetStringValue() string {

	// binary string encoding of s.v without any leading 0s
	res := strconv.FormatUint(s.v, 2)

	// append any leading 0s if needed
	n := s.d - uint(len(res))
	for n > 0 {
		res = "0" + res
		n--
	}

	return res
}

// Get the binary value encoded in the string. e.g. 12
func (s *SMBinaryString) GetValue() uint64 {
	return s.v
}

// return number of digits including leading 0s if any
func (s *SMBinaryString) GetDigitsCount() uint {
	return s.d
}
