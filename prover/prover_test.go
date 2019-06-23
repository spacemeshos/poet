package prover

import (
	"fmt"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetProof(t *testing.T) {
	r := require.New(t)

	challenge := []byte("challenge this")
	merkleProof, err := GetProof(hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), 16, 5)
	r.NoError(err)
	fmt.Printf("root: %x\n", merkleProof.Root)
	fmt.Printf("proof: %x\n", merkleProof.ProvenLeaves)
}

func BenchmarkGetProof(b *testing.B) {
	r := require.New(b)

	challenge := []byte("challenge this! challenge this! ")
	leafCount := uint64(1) << 20
	securityParam := shared.T
	fmt.Printf("=> Generating proof for %d leaves with security param %d...\n", leafCount, securityParam)

	t1 := time.Now()
	_, err := GetProof(hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafCount, securityParam)
	e := time.Since(t1)

	r.NoError(err)
	fmt.Printf("=> Completed in %v.\n", e)

	/*
		=> Generating proof for 1048576 leaves with security param 150...
		=> Completed in 22.020794606s.
	*/
}
