package internal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
	"math/big"
)

type Challenge = shared.Challenge
type Proof = shared.Proof
type IVerifier = shared.IBasicVerifier
type Identifier = shared.Identifier
type HashFunc = shared.HashFunc

type SMVerifier struct {
	x []byte   // commitment
	n uint     // n param 1 <= n <= 63
	h HashFunc // Hx()
}

// Verify proof p for challenge c.
// Returns true iff the proof is verified, false otherwise
func (s *SMVerifier) Verify(c Challenge, p Proof) bool {

	// We use k,v memory storage to store label values for identifiers
	// as they are unique in p
	m := make(map[string][shared.WB]byte)
	f := NewSMBinaryStringFactory()

	// iterate over each identifier in the challenge and verify the proof for it
	for idx, id := range c.Data {

		bs, err := f.NewBinaryString(string(id))
		if err != nil {
			return false
		}

		siblingIds, err := bs.GetBNSiblings()
		if err != nil {
			return false
		}

		// a slice of all labels included in proof
		proofLabels := p.L[idx][:]

		// first label in the list if the leaf (node id) label - read it and remove it from the slice
		// we use labelValue as the label of the node on the path from the leaf to the root, including both leaf and root
		labelValue, proofLabels := proofLabels[0], proofLabels[1:]

		for _, siblingId := range siblingIds { // siblings ids up the path from the leaf to the root

			sibId := siblingId.GetStringValue()
			var sibValue shared.Label
			var ok bool
			if sibValue, ok = m[sibId]; !ok { // label is not the k/v mem store - read it from the proof and store it in the k/v store
				// take label from the head of the slice and remove it from the slice
				sibValue, proofLabels = proofLabels[0], proofLabels[1:]
				m[sibId] = sibValue
			}

			// calculate the label of the next node up the path to root, based on its parent nodes labels
			parentNodeId, err := siblingId.TruncateLSB()
			if err != nil {
				return false
			}

			// pack data to hash
			labelData := append([]byte(parentNodeId.GetStringValue()), sibValue[:]...)
			labelData = append(labelData, labelValue[:]...)

			// hx(siblingPartentNodeId, siblingLabel, currentNodeOnPathValue)
			labelValue = s.h.Hash(labelData)
		}

		// labelValue should be equal to the root label provided by the proof
		if bytes.Compare(labelValue[:], p.Phi[:]) != 0 {
			return false
		}
	}
	return true
}

// γ := (Hx(φ,1),...Hx(φ,t))
func (s *SMVerifier) CreteNipChallenge(phi []byte) (Challenge, error) {

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
		hash := s.h.Hash(d)

		// Take the first 64 bits from the hash
		bg := new(big.Int).SetBytes(hash[0:8])

		// Integer representation of the first 64 bits
		v := bg.Uint64()

		// encode v as a 64 bits binary string
		bs, err := f.NewBinaryStringFromInt(v, 64)
		str := bs.GetStringValue()

		// take the first s.n bits from the 64 bits binary string
		l := uint(len(str))
		if l > s.n {
			str = str[0:s.n]
		}

		data[i] = Identifier(str)
	}

	return Challenge{Data: data}, nil
}

// create a random challenge that can be used to challenge a prover
// that created a proof for shared params (x,t,n,w)
func (s *SMVerifier) CreteRndChallenge() (Challenge, error) {

	var data [shared.T]Identifier
	f := NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		b, err := f.NewRandomBinaryString(s.n)
		if err != nil {
			return Challenge{}, err
		}
		data[i] = Identifier(b.GetStringValue())
	}

	c := Challenge{Data: data}
	return c, nil
}

// Create a new verifier for commitment X and param n
func NewVerifier(x []byte, n uint) (IVerifier, error) {

	if n < 1 || n > 63 {
		return nil, errors.New("n must be in range [1, 63]")
	}

	res := &SMVerifier{
		x: x,
		n: n,
		h: shared.NewHashFunc(x),
	}

	return res, nil
}
