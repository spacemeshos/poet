package prover

import (
	"fmt"
	"github.com/spacemeshos/poet/hash"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetProof(t *testing.T) {
	r := require.New(t)

	challenge := []byte("challenge this")
	merkleProof, err := GetProof(hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), 16, 5)
	r.NoError(err)
	fmt.Printf("root: %x\n", merkleProof.Root)
	fmt.Printf("proof: %x\n", merkleProof.ProvenLeaves)
}
