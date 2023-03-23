package shared

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"

	"github.com/spacemeshos/go-scale/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiatShamir(t *testing.T) {
	challenge := []byte("challenge this")

	occurrences := make(map[uint64]uint)
	rounds := 5000
	indicesPerRound := uint8(255)
	spaceSize := uint64(30000000000)
	buckets := 10
	bucketDivisor := spaceSize / uint64(buckets)
	expectedIndicesPerBucket := rounds * int(indicesPerRound) / buckets
	for i := 0; i < rounds; i++ {
		binary.BigEndian.PutUint64(challenge, uint64(i))
		// println(i, string(challenge))
		shamir := FiatShamir(challenge, spaceSize, indicesPerRound)
		for key := range shamir {
			occurrences[key/bucketDivisor]++
		}
	}
	// fmt.Println("expected indices per bucket:", expectedIndicesPerBucket)
	for i := 0; i < buckets; i++ {
		deviationFromExpected := float64(int(occurrences[uint64(i)])-expectedIndicesPerBucket) /
			float64(expectedIndicesPerBucket)
		// fmt.Printf("%d %d %+0.3f%%\n", i, occurrences[uint64(i)], 100*deviationFromExpected)
		assert.True(t, math.Abs(deviationFromExpected) < 0.005,
			"deviation from expected cannot exceed 0.5%% (for bucket %d it was %+0.3f%%)", i,
			100*deviationFromExpected)
	}
}

func TestFiatShamirReturnsCorrectNumOfIndices(t *testing.T) {
	challenge := []byte("challenge")

	// returns requested number of indices
	require.Len(t, FiatShamir(challenge, 10, 5), 5)

	// limits index count to space size (otherwise an infinite loop would occur)
	require.Len(t, FiatShamir(challenge, 10, 15), 10)

	// if the space is big enough, return the requested index count
	require.Len(t, FiatShamir(challenge, 20, 15), 15)
}

// This test used to hang.
// See https://github.com/spacemeshos/poet/issues/173
func TestFiatShamirLowSpaceSize(t *testing.T) {
	FiatShamir([]byte("challenge this"), uint64(T)+1, T)
}

func TestMakeLabel(t *testing.T) {
	r := require.New(t)
	stringHash := func(data []byte) []byte {
		return []byte("H(" + hex.EncodeToString(data) + ")")
	}

	decodeHexString := func(str string) []byte {
		if str == "" {
			return nil
		}
		bytes, err := hex.DecodeString(str)
		r.NoError(err)
		return bytes
	}

	makeSiblings := func(strings ...string) [][]byte {
		var ret [][]byte
		for _, str := range strings {
			ret = append(ret, decodeHexString(str))
		}
		return ret
	}

	makeLabel := MakeLabelFunc()

	r.Equal("H(0000000000000001)",
		string(makeLabel(stringHash, 1, makeSiblings())))

	r.Equal("H(000000000000000211111111)",
		string(makeLabel(stringHash, 2, makeSiblings("11111111"))))

	r.Equal("H(00000000000000031111111133333333)",
		string(makeLabel(stringHash, 3, makeSiblings("11111111", "", "33333333"))))
}

func FuzzMerkleProofConsistency(f *testing.F) {
	f.Add([]byte("018912380012"))
	tester.FuzzConsistency[MerkleProof](f)
}

func FuzzMerkleProofSafety(f *testing.F) {
	f.Add([]byte("018912380012"))
	tester.FuzzSafety[MerkleProof](f)
}

func BenchmarkFiatShamir(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FiatShamir([]byte("challenge this"), uint64(30000000000), T)
	}
}
