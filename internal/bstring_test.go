package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBinaryString(t *testing.T) {

	const d = "00010001011010010001100111000010010101101111111111101110000110"
	f := NewSMBinaryStringFactory()
	b, err := f.NewBinaryString(d)

	assert.NoError(t, err)
	assert.Equal(t, uint(len(d)), b.GetDigitsCount())
	assert.Equal(t, d, b.GetStringValue())
}

func TestRandomBinaryString(t *testing.T) {
	f := NewSMBinaryStringFactory()
	for i := 0; i < 10000; i++ {
		const d = uint(63)
		b, err := f.NewRandomBinaryString(d)
		assert.NoError(t, err)
		assert.Equal(t, uint(d), b.GetDigitsCount())
	}
}

func TestInvalidBinaryString(t *testing.T) {
	f := NewSMBinaryStringFactory()
	// try to create a binary string from an invalid string - contains chars other than 0 and 1
	_, err := f.NewBinaryString("0102")
	assert.Error(t, err)
}

func TestInvalidBinaryStringFromInt(t *testing.T) {
	f := NewSMBinaryStringFactory()
	// try to create from int with insufficient number of digits to encode int value to bits
	_, err := f.NewBinaryStringFromInt(1025, 5)
	assert.Error(t, err)
}

func TestBinaryStringFromInt(t *testing.T) {
	f := NewSMBinaryStringFactory()

	// 126 decimal is 1111101, requiring 7 binary digits
	v := uint64(125)

	b, err := f.NewBinaryStringFromInt(v, 7)
	assert.NoError(t, err)
	//println(b.GetStringValue())
	assert.Equal(t, uint(7), b.GetDigitsCount())
	assert.Equal(t, v, b.GetValue())

	// encode to an 8 bits binary string
	b, err = f.NewBinaryStringFromInt(v, 8)
	assert.NoError(t, err)
	//println(b.GetStringValue())
	assert.Equal(t, uint(8), b.GetDigitsCount())
	assert.Equal(t, v, b.GetValue())

	// encode to a 63 bits binary string
	b, err = f.NewBinaryStringFromInt(v, 63)
	assert.NoError(t, err)
	//println(b.GetStringValue())
	assert.Equal(t, uint(63), b.GetDigitsCount())
	assert.Equal(t, v, b.GetValue())
}