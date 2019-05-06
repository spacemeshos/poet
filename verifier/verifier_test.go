package verifier

import (
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidate(t *testing.T) {
	r := require.New(t)

	challenge := shared.Sha256Challenge("challenge")
	leafCount := uint64(16)
	securityParam := uint8(4)
	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	r.NoError(err)

	err = Validate(merkleProof, challenge, leafCount, securityParam)
	r.NoError(err)
}

func TestValidateWrongSecParam(t *testing.T) {
	merkleProof := shared.MerkleProof{
		Root:         nil,
		ProvenLeaves: [][]byte{nil, nil},
		ProofNodes:   nil,
	}
	challenge := shared.Sha256Challenge("challenge")
	leafCount := uint64(16)
	securityParam := uint8(4)
	err := Validate(merkleProof, challenge, leafCount, securityParam)
	require.EqualError(t, err, "number of proven leaves (2) must be equal to security param (4)")
}

func TestValidateWrongMerkleValidationError(t *testing.T) {
	merkleProof := shared.MerkleProof{
		Root:         nil,
		ProvenLeaves: [][]byte{},
		ProofNodes:   nil,
	}
	challenge := shared.Sha256Challenge("challenge")
	leafCount := uint64(16)
	securityParam := uint8(0)
	err := Validate(merkleProof, challenge, leafCount, securityParam)
	require.EqualError(t, err, "error while validating merkle proof: at least one leaf is required for validation")
}

func TestValidateWrongRoot(t *testing.T) {
	r := require.New(t)

	challenge := shared.Sha256Challenge("challenge")
	leafCount := uint64(16)
	securityParam := uint8(4)
	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	r.NoError(err)

	merkleProof.Root[0] = 0

	err = Validate(merkleProof, challenge, leafCount, securityParam)
	r.EqualError(err, "merkle proof not valid")
}

type badHashChallenge []byte

func (c badHashChallenge) MerkleHashFunc() merkle.HashFunc {
	return shared.Sha256Challenge(c).MerkleHashFunc()
}

func (c badHashChallenge) LabelHashFunc() shared.LabelHashFunc {
	return func(data []byte) []byte {
		return []byte("not the right thing!")
	}
}

func TestValidateFailLabelValidation(t *testing.T) {
	r := require.New(t)

	challenge := shared.Sha256Challenge("challenge")
	leafCount := uint64(16)
	securityParam := uint8(4)
	merkleProof, err := prover.GetProof(challenge, leafCount, securityParam)
	r.NoError(err)

	err = Validate(merkleProof, badHashChallenge(challenge), leafCount, securityParam)
	r.Error(err)
	r.Regexp("label at index 0 incorrect - expected: [0-f]* actual: [0-f]*", err.Error())
}
