package hash

import (
	"github.com/stretchr/testify/require"
	"testing"
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
	r.Equal(GenMerkleHashFunc(aChallenge)(lChild, rChild), GenMerkleHashFunc(aChallenge)(lChild, rChild))

	// different challenge -> different hash
	r.NotEqual(GenMerkleHashFunc(aChallenge)(lChild, rChild), GenMerkleHashFunc(bChallenge)(lChild, rChild))

	// different children (e.g. different order) -> different hash
	r.NotEqual(GenMerkleHashFunc(aChallenge)(lChild, rChild), GenMerkleHashFunc(aChallenge)(rChild, lChild))
}
