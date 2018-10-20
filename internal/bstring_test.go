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

func TestFlipLSB(t *testing.T) {

	const d = "101011100110000111110011101010101110111101000000001011110101110"
	f := NewSMBinaryStringFactory()

	b, err := f.NewBinaryString(d)
	assert.NoError(t, err)

	b1, err := b.FlipLSB()
	assert.NoError(t, err)

	s := b.GetStringValue()
	s1 := b1.GetStringValue()
	assert.NotEqual(t, s[len(s)-1], s1[len(s1)-1])
}

func TestTruncateLSB(t *testing.T) {

	const d = "101011100110000111110011101010101110111101000000001011110101110"
	f := NewSMBinaryStringFactory()

	b, err := f.NewBinaryString(d)
	assert.NoError(t, err)

	b1, err := b.TruncateLSB()
	assert.NoError(t, err)

	s := b.GetStringValue()
	s1 := b1.GetStringValue()
	assert.Equal(t, len(s), len(s1)+1)

	// reconstruct s by appending s[LSB] to s1 - the truncated string
	s3 := s1 + s[len(s)-1:]
	assert.Equal(t, s, s3)

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