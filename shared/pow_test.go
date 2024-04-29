package shared_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
)

func TestCheckLeadingZeroBits(t *testing.T) {
	r := require.New(t)

	r.True(shared.CheckLeadingZeroBits([]byte{0x00}, 0))
	r.True(shared.CheckLeadingZeroBits([]byte{0x00}, 8))

	// Out of bounds
	r.False(shared.CheckLeadingZeroBits([]byte{0x00}, 9))

	r.True(shared.CheckLeadingZeroBits([]byte{0x0F}, 4))
	r.False(shared.CheckLeadingZeroBits([]byte{0x0F}, 5))

	r.True(shared.CheckLeadingZeroBits([]byte{0x00, 0x0F}, 5))
	r.True(shared.CheckLeadingZeroBits([]byte{0x00, 0x0F}, 12))
	r.False(shared.CheckLeadingZeroBits([]byte{0x00, 0x0F}, 13))
}

func BenchmarkPowHash(b *testing.B) {
	powChallenge := make([]byte, 32)
	poetChallenge := make([]byte, 32)
	nodeID := make([]byte, 32)

	var hash []byte
	p := shared.NewPowHasher(powChallenge, poetChallenge, nodeID)
	for i := 0; i < b.N; i++ {
		hash = p.Hash(123678, hash[:0])
	}
}

func BenchmarkFindSubmitPowNonce(b *testing.B) {
	powChallenge := make([]byte, 32)
	poetChallenge := make([]byte, 32)
	nodeID := make([]byte, 32)
	b.ResetTimer()
	for difficulty := 20; difficulty <= 20; difficulty += 1 {
		b.Run(fmt.Sprintf("difficulty=%v", difficulty), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				binary.LittleEndian.PutUint64(nodeID, uint64(i))
				_, err := shared.FindSubmitPowNonce(
					context.Background(),
					powChallenge,
					poetChallenge,
					nodeID,
					uint(difficulty),
				)
				require.NoError(b, err)
			}
		})
	}
}
