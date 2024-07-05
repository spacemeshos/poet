package hash

import (
	"testing"

	"github.com/spacemeshos/sha256-simd"
	"github.com/stretchr/testify/require"
)

func TestGenLabelHashFunc(t *testing.T) {
	t.Parallel()

	aChallenge, bChallenge := []byte("a"), []byte("b")
	data, other := []byte("data"), []byte("other")

	// same challenge and data -> same hash
	aHash := GenLabelHashFunc(aChallenge)(data)
	require.Equal(t, aHash, GenLabelHashFunc(aChallenge)(data))

	// different challenge -> different hash
	require.NotEqual(t, aHash, GenLabelHashFunc(bChallenge)(data))

	// different data -> different hash
	require.NotEqual(t, aHash, GenLabelHashFunc(aChallenge)(other))
}

func TestGenMerkleHashFunc(t *testing.T) {
	t.Parallel()

	aChallenge, bChallenge := []byte("a"), []byte("b")
	lChild, rChild := []byte("l"), []byte("r")

	// same challenge and children -> same hash
	aHash := GenMerkleHashFunc(aChallenge)(nil, lChild, rChild)
	require.Equal(t, aHash, GenMerkleHashFunc(aChallenge)(nil, lChild, rChild))

	// different challenge -> different hash
	require.NotEqual(t, aHash, GenMerkleHashFunc(bChallenge)(nil, lChild, rChild))

	// different children (e.g. different order) -> different hash
	require.NotEqual(t, aHash, GenMerkleHashFunc(aChallenge)(nil, rChild, lChild))
}

func TestGenLabelHashFuncHash(t *testing.T) {
	t.Parallel()
	challenge := []byte("a123123123mskofvuw098e71bnj91273")
	data := []byte("12312390819023879012379184718920408data0123128991231239081")

	expected := []byte{
		0xb0, 0xd5, 0x37, 0x8b, 0x51, 0x46, 0x33, 0x92, 0xb2, 0x93, 0x47, 0xb6, 0x86, 0xd8, 0xa7,
		0xf0, 0x41, 0xce, 0xe, 0xc2, 0x78, 0xdd, 0x3e, 0x9, 0xc, 0x4e, 0x4c, 0xe0, 0x5e, 0x61, 0xc4, 0xc4,
	}

	t.Run("generic implementation", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			require.Equal(t, expected, genLabelHashFuncGeneric(challenge)(data))
		}
	})
	t.Run("minio implementation", func(t *testing.T) {
		t.Parallel()
		if _, ok := sha256.New().(*sha256.Digest); !ok {
			t.Skip("SHA extensions are not supported")
		}
		for i := 0; i < 10; i++ {
			require.Equal(t, expected, genLabelHashFuncMinio(challenge)(data))
		}
	})
	t.Run("public hash func generator", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			require.Equal(t, expected, GenLabelHashFunc(challenge)(data))
		}
	})
}

func TestGenMekleHashFuncHash(t *testing.T) {
	t.Parallel()
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
			require.Equal(t, expected, digest)
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
			require.Equal(t, expected, digest)
		}
	})
	t.Run("public hash func generator", func(t *testing.T) {
		t.Parallel()
		var digest []byte
		for i := 0; i < 10; i++ {
			digest = GenMerkleHashFunc(challenge)(digest, lChild, rChild)
			require.Equal(t, expected, digest)
		}
	})
}
