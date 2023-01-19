package hash

import (
	"testing"

	"github.com/spacemeshos/sha256-simd"
	"github.com/stretchr/testify/require"
)

func TestGenLabelHashFunc(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	req := require.New(t)
	challenge := []byte("a123123123mskofvuw098e71bnj91273")
	data := []byte("12312390819023879012379184718920408data0123128991231239081")

	expected := []byte{
		104, 1, 230, 180, 3, 202, 139, 10, 164, 69, 74, 169, 116, 77, 27, 134,
		138, 163, 204, 32, 182, 197, 205, 194, 13, 48, 30, 224, 254, 1, 46, 133,
	}

	t.Run("generic implementation", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			req.Equal(expected, genLabelHashFuncGeneric(challenge)(data))
		}
	})
	t.Run("minio implementation", func(t *testing.T) {
		t.Parallel()
		if _, ok := sha256.New().(*sha256.Digest); !ok {
			t.Skip("SHA extensions are not supported")
		}
		for i := 0; i < 10; i++ {
			req.Equal(expected, genLabelHashFuncMinio(challenge)(data))
		}
	})
	t.Run("public hash func generator", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			req.Equal(expected, GenLabelHashFunc(challenge)(data))
		}
	})
}

func TestGenMekleHashFuncHash(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	challenge := []byte("a123123123mskofvuw098e71bnj91273")
	lChild, rChild := []byte("left-one"), []byte("right-one")

	expected := []byte{
		103, 1, 150, 181, 115, 169, 147, 120, 81, 61, 244, 238, 253, 189, 132, 180,
		181, 155, 232, 100, 130, 168, 223, 56, 2, 186, 250, 51, 76, 118, 6, 41,
	}

	t.Run("generic implementation", func(t *testing.T) {
		t.Parallel()
		var digest []byte
		for i := 0; i < 10; i++ {
			digest = genMerkleHashFuncGeneric(challenge)(digest, lChild, rChild)
			req.Equal(expected, digest)
		}
	})
	t.Run("minio implementation", func(t *testing.T) {
		t.Parallel()
		if _, ok := sha256.New().(*sha256.Digest); !ok {
			t.Skip("SHA extensions are not supported")
		}
		var digest []byte
		for i := 0; i < 10; i++ {
			digest = genMerkleHashFuncMinio(challenge)(digest, lChild, rChild)
			req.Equal(expected, digest)
		}
	})
	t.Run("public hash func generator", func(t *testing.T) {
		t.Parallel()
		var digest []byte
		for i := 0; i < 10; i++ {
			digest = GenMerkleHashFunc(challenge)(digest, lChild, rChild)
			req.Equal(expected, digest)
		}
	})
}
