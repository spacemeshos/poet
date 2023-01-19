package hash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenLabelHashFunc(t *testing.T) {
	r := require.New(t)

	aChallenge, bChallenge := []byte("a"), []byte("b")
	data, other := []byte("data"), []byte("other")

	// same challenge and data -> same hash
	r.Equal(GenLabelHashFunc(aChallenge)(data), GenLabelHashFunc(aChallenge)(data))

	// different challenge -> different hash
	r.NotEqual(GenLabelHashFunc(aChallenge)(data), GenLabelHashFunc(bChallenge)(data))

	// different data -> different hash
	r.NotEqual(GenLabelHashFunc(aChallenge)(data), GenLabelHashFunc(aChallenge)(other))
}

func TestGenMerkleHashFunc(t *testing.T) {
	r := require.New(t)

	aChallenge, bChallenge := []byte("a"), []byte("b")
	lChild, rChild := []byte("l"), []byte("r")

	// same challenge and children -> same hash
	r.Equal(GenMerkleHashFunc(aChallenge)(nil, lChild, rChild), GenMerkleHashFunc(aChallenge)(nil, lChild, rChild))

	// different challenge -> different hash
	r.NotEqual(GenMerkleHashFunc(aChallenge)(nil, lChild, rChild), GenMerkleHashFunc(bChallenge)(nil, lChild, rChild))

	// different children (e.g. different order) -> different hash
	r.NotEqual(GenMerkleHashFunc(aChallenge)(nil, lChild, rChild), GenMerkleHashFunc(aChallenge)(nil, rChild, lChild))
}

func TestGenLabelHashFuncHash(t *testing.T) {
	challenge := []byte("a123123123mskofvuw098e71bnj91273")
	data := []byte("12312390819023879012379184718920408data0123128991231239081")

	require.Equal(
		t,
		[]byte{
			104, 1, 230, 180, 3, 202, 139, 10, 164, 69, 74, 169, 116, 77, 27, 134,
			138, 163, 204, 32, 182, 197, 205, 194, 13, 48, 30, 224, 254, 1, 46, 133,
		},
		GenLabelHashFunc(challenge)(data),
	)
}

func TestGenMekleHashFuncHash(t *testing.T) {
	challenge := []byte("a123123123mskofvuw098e71bnj91273")
	lChild, rChild := []byte("left-one"), []byte("right-one")

	require.Equal(
		t,
		[]byte{
			103, 1, 150, 181, 115, 169, 147, 120, 81, 61, 244, 238, 253, 189, 132, 180,
			181, 155, 232, 100, 130, 168, 223, 56, 2, 186, 250, 51, 76, 118, 6, 41,
		},
		GenMerkleHashFunc(challenge)(nil, lChild, rChild),
	)
}
