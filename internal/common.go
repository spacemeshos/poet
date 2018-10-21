package internal

import (
	"bytes"
	"encoding/binary"
	"github.com/spacemeshos/poet-ref/shared"
	"math/big"
)

// Shared logic between reference verifier and prover

// Both prover and verifier need to be able to create a nip challenge
// phi - root label
// h - Hx()
// n - proof param
func creteNipChallenge(phi []byte, h HashFunc, n uint) (Challenge, error) {

	var data [shared.T]Identifier
	buf := new(bytes.Buffer)
	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		buf.Reset()
		err := binary.Write(buf, binary.BigEndian, uint8(i))
		if err != nil {
			return Challenge{}, err
		}

		// pack (phi, i) into a bytes array
		// Hx(phi, i) := Hx(phi ... bigEndianEncodedBytes(i))
		d := append(phi, buf.Bytes()...)

		// Compute Hx(phi, i)
		hash := h.Hash(d)

		// Take the first 64 bits from the hash
		bg := new(big.Int).SetBytes(hash[0:8])

		// Integer representation of the first 64 bits
		v := bg.Uint64()

		// encode v as a 64 bits binary string
		bs, err := f.NewBinaryStringFromInt(v, 64)
		str := bs.GetStringValue()

		// take the first s.n bits from the 64 bits binary string
		l := uint(len(str))
		if l > n {
			str = str[0:n]
		}

		data[i] = Identifier(str)
	}

	return Challenge{Data: data}, nil
}
