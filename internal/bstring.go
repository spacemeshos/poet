package internal

import (
	"github.com/spacemeshos/poet-ref/shared"

	"fmt"
	"strconv"
)

type SMBinaryString struct {
	s string
}

func NewBinaryString(v string) shared.BinaryString {
	return &SMBinaryString {
		s: v,
	}
}

func NewBinaryStringFromInt(v uint64, digits int) shared.BinaryString {

	s := strconv.FormatUint(v, 2)

	// add leading zeros if needed to s
	// todo: profile and optimize
	n := digits - len(s)
	for n > 0 {
		s = "0" + s
		n--
	}

	return &SMBinaryString {
		s: s,
	}
}

// Get string representation. e.g. "00011"
func (s* SMBinaryString) GetStringValue() string {
	return s.s
}

// Get the binary value encoded in the string. e.g. 12
func (s* SMBinaryString) GetBinaryValue() (uint64, error) {

	res, err := strconv.ParseUint(s.s, 2, 64)
	if err != nil {
		fmt.Println("Invalid binary string: ", err)
		return 0, err
	}

	return res, nil
}

// return number of digits including leading 0s if any
func (s* SMBinaryString) GetDigitsCount() uint {
	return uint(len(s.s))
}