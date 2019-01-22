package internal

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
)

const EmptyString = ""

type SMBinaryStringFactory struct {
	// cache map[uint64] map[uint]*SMBinaryString
	// cache1 map[string] *SMBinaryString
}

func NewSMBinaryStringFactory() BinaryStringFactory {
	return &SMBinaryStringFactory{
		// make(map[uint64] map[uint]*SMBinaryString, 500),
		// make(map[string]*SMBinaryString, 500),
	}
}

type SMBinaryString struct {
	v uint64 // stored value
	d uint   // number of binary digits to display
	f *SMBinaryStringFactory
}

// digits must be at least as large to represent v
func (f *SMBinaryStringFactory) NewBinaryStringFromInt(v uint64, d uint) (BinaryString, error) {

	// todo: only test in debug builds but not in production ones

	/*
		l := uint(bits.Len64(v))
		if l > d {
			return nil, errors.New("invalid digits. Digits must be large enough to represent v in bits")
		}*/

	res := &SMBinaryString{
		v: v,
		d: d,
		f: f,
	}

	return res, nil
}

// Create a new BinaryString from a string of 0s and 1s, e.g. "00111"
// Returns an error if s is not a valid binary string, e.g. it contains chars
// other then 0 or 1
// Any leading 0s will be included in the result
func (f *SMBinaryStringFactory) NewBinaryString(s string) (BinaryString, error) {

	var v uint64

	if s != EmptyString {
		parsed, err := strconv.ParseUint(s, 2, 64)
		if err != nil {
			return nil, err
		}
		v = parsed
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
func (f *SMBinaryStringFactory) NewRandomBinaryString(d uint) (BinaryString, error) {

	if d > 63 {
		return nil, errors.New("unsupported # of digits. must be less or equals to 64")
	}

	if d == 0 { // the only id with 0 digits is ""
		return f.NewBinaryString(EmptyString)
	}

	// generate a random number with d digits

	// compute 2^d
	max := uint64(math.Pow(2, float64(d)))
	//max := uint64(math.Exp2(float64(d)))

	maxBig := new(big.Int).SetUint64(max)

	// max int with d digits is 2^d - 1. The following returns rnd in range [0...2^d-1]
	rndBig, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		return nil, err
	}

	v := rndBig.Uint64()
	return f.NewBinaryStringFromInt(v, d)
}

// returns list of siblings on the path from s the root assuming s is a node identifier in a full binary tree
func (s *SMBinaryString) GetBNSiblings(leftOnly bool) ([]BinaryString, error) {

	// slice of siblings
	var res []BinaryString

	if s.v == 0 && s.d == 0 { // special case - dag root node
		return res, nil
	}

	// current node pointer
	var nodeId BinaryString

	// initial value
	nodeId = s

	for {

		// append node's sibling to result
		siblingNode, err := nodeId.FlipLSB()
		if err != nil {
			return nil, err
		}

		if !leftOnly || siblingNode.IsEven() {
			// we add to results if caller didn't request leftOnly
			// or she did and the sibling is a left sibling (LSB == '0')
			res = append(res, siblingNode)
		}

		// println("Adding sibling: ", siblingNode.GetStringValue())

		// continue with the node's parent node
		nodeId, err = nodeId.TruncateLSB()

		if err != nil {
			return nil, err
		}

		if nodeId.GetDigitsCount() == 0 {
			break
		}
	}

	if len(res) == 0 && !leftOnly {
		return res, errors.New("expected one or more siblings on the path to root")
	}

	return res, nil
}

// Returns a new BinaryString with the LSB truncated. e.g. "0010" => "001"
func (s *SMBinaryString) TruncateLSB() (BinaryString, error) {
	return s.f.NewBinaryStringFromInt(s.v>>1, s.d-1)
}

// Flip LSB. e.g. "0010" => "0011"
func (s *SMBinaryString) FlipLSB() (BinaryString, error) {
	return s.f.NewBinaryStringFromInt(s.v^1, s.d)
}

// Get string representation. e.g. "00011"
func (s *SMBinaryString) GetStringValue() string {

	if s.d == 0 {
		// special case - empty binary string
		return EmptyString
	}

	// binary string encoding of s.v without any leading 0s
	res := strconv.FormatUint(s.v, 2)

	// prepend any leading 0s if needed
	n := s.d - uint(len(res))
	if n == 0 {
		return res
	}

	// use a string build to prepend 0s
	var b strings.Builder
	for n > 0 {
		b.WriteString("0")
		n--
	}
	// append the values
	b.WriteString(res)

	return b.String()
}

// Get the binary value encoded in the string. e.g. 12
func (s *SMBinaryString) GetValue() uint64 {
	return s.v
}

func (s *SMBinaryString) IsEven() bool {
	return s.v%2 == 0
}

func (s *SMBinaryString) IsOdd() bool {
	return !s.IsEven()
}

// return number of digits including leading 0s if any
func (s *SMBinaryString) GetDigitsCount() uint {
	return s.d
}
