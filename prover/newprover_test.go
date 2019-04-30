package prover

import (
	"fmt"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetProof(t *testing.T) {
	r := require.New(t)

	challenge := shared.Sha256Challenge("challenge this")
	merkleProof, err := GetProof(challenge, 4)
	r.NoError(err)
	fmt.Printf("root: %x\n", merkleProof.root)
	fmt.Printf("proof: %x\n", merkleProof.provenLeaves)
}
