package shared

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSha256Challenge_LabelHashFunc(t *testing.T) {
	r := require.New(t)

	aChallenge, bChallenge := Sha256Challenge("a"), Sha256Challenge("b")
	data, other := []byte("data"), []byte("other")

	// same challenge and data -> same hash
	r.Equal(aChallenge.LabelHashFunc()(data), aChallenge.LabelHashFunc()(data))

	// different challenge -> different hash
	r.NotEqual(aChallenge.LabelHashFunc()(data), bChallenge.LabelHashFunc()(data))

	// different data -> different hash
	r.NotEqual(aChallenge.LabelHashFunc()(data), aChallenge.LabelHashFunc()(other))
}

func TestSha256Challenge_MerkleHashFunc(t *testing.T) {
	r := require.New(t)

	aChallenge, bChallenge := Sha256Challenge("a"), Sha256Challenge("b")
	lChild, rChild := []byte("l"), []byte("r")

	// same challenge and children -> same hash
	r.Equal(aChallenge.MerkleHashFunc()(lChild, rChild), aChallenge.MerkleHashFunc()(lChild, rChild))

	// different challenge -> different hash
	r.NotEqual(aChallenge.MerkleHashFunc()(lChild, rChild), bChallenge.MerkleHashFunc()(lChild, rChild))

	// different children (e.g. different order) -> different hash
	r.NotEqual(aChallenge.MerkleHashFunc()(lChild, rChild), aChallenge.MerkleHashFunc()(rChild, lChild))
}
