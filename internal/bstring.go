package internal

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math/bits"
	"strconv"
)

type SMBinaryString struct {
	v uint64  	// stored value
	d uint		// number of binary digits to display
}

// Create a new BinaryString from a string of 0s and 1s, e.g. "00111"
// Returns an error if s is not a valid binary string, e.g. it contains chars
// other then 0 or 1
// Any leading 0s will be included in the result
func NewBinaryString(s string) (shared.BinaryString, error) {

	v, err := strconv.ParseUint(s, 2, 64)
	if err != nil {
		return nil, err
	}

	res := &SMBinaryString {
		v: v,
		d: uint(len(s)),
	}

	return res, nil
}

// Create a new BinaryString digits long. e.g for digits=4 "0110"
func NewRandomBinaryString(digits uint) (shared.BinaryString, error) {

	// generate l random bytes
	var l = digits / 8
	var extra = digits % 8
	if extra != 0 {
		l++
	}

	buff := make([]byte, l)
	_, err := rand.Read(buff)
	if err != nil {
		return nil, err
	}
	
	// todo: zero the first (8 - extra) bits in the first byte
	// 0 1 2 3 4 5 6 7
	// 0 0 0
	// msb := buff[0]

	v := binary.BigEndian.Uint64(buff)

	// create a binary string
	b, err := NewBinaryStringFromInt(v, digits)

	return b, err
}

// digits must be at least as large to represent v
func NewBinaryStringFromInt(v uint64, digits uint) (shared.BinaryString,error) {

	l := uint(bits.Len64(v))
	if l > digits {
		return nil, errors.New("invalid digits. Digits must be large enough to reprsent v in bits")
	}

	res := &SMBinaryString{
		v: v,
		d: digits,
	}

	return res, nil
}

// Get string representation. e.g. "00011"
func (s* SMBinaryString) GetStringValue() string {

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
func (s* SMBinaryString) GetValue() uint64 {
	return s.v
}

// return number of digits including leading 0s if any
func (s* SMBinaryString) GetDigitsCount() uint {
	return s.d
}