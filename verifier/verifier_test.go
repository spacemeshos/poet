package verifier

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
)

func testValidate(t *testing.T, minMemoryLayer uint) {
	r := require.New(t)
	challenge := []byte("challenge")
	securityParam := uint8(1)
	leaves, merkleProof, err := prover.GenerateProofWithoutPersistency(
		context.Background(),
		prover.TreeConfig{
			MinMemoryLayer: minMemoryLayer,
			Datadir:        t.TempDir(),
		},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(100*time.Millisecond),
		securityParam,
	)
	r.NoError(err)
	err = Validate(
		*merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		leaves,
		securityParam,
	)
	r.NoError(err, "leaves %d", leaves)
}

func TestValidate(t *testing.T) {
	t.Run("Leaves", func(t *testing.T) {
		testValidate(t, 0)
	})
	t.Run("NoLeaves", func(t *testing.T) {
		testValidate(t, prover.LowestMerkleMinMemoryLayer)
	})
}

func TestValidateWrongSecParam(t *testing.T) {
	merkleProof := shared.MerkleProof{
		Root:         nil,
		ProvenLeaves: make([][]byte, 2),
		ProofNodes:   nil,
	}
	challenge := []byte("challenge")
	numLeaves := uint64(16)
	securityParam := uint8(4)
	err := Validate(
		merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		numLeaves,
		securityParam,
	)
	require.EqualError(t, err, "number of proven leaves (2) must be equal to security param (4)")
}

func TestValidateWrongMerkleValidationError(t *testing.T) {
	merkleProof := shared.MerkleProof{
		Root:         nil,
		ProvenLeaves: make([][]byte, 0),
		ProofNodes:   nil,
	}
	challenge := []byte("challenge")
	numLeaves := uint64(16)
	securityParam := uint8(0)
	err := Validate(
		merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		numLeaves,
		securityParam,
	)
	require.EqualError(t, err, "error while validating merkle proof: at least one leaf is required for validation")
}

func TestValidateWrongRoot(t *testing.T) {
	r := require.New(t)

	challenge := []byte("challenge")
	duration := 100 * time.Millisecond
	securityParam := uint8(4)
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(
		context.Background(),
		prover.TreeConfig{Datadir: t.TempDir()},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(duration),
		securityParam,
	)
	r.NoError(err)

	merkleProof.Root[0] = 0

	err = Validate(
		*merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		leafs,
		securityParam,
	)
	r.EqualError(err, "merkle proof not valid")
}

func BadLabelHashFunc(data []byte) []byte {
	return []byte("not the right thing!")
}

func TestValidateFailLabelValidation(t *testing.T) {
	r := require.New(t)

	challenge := []byte("challenge")
	duration := 100 * time.Millisecond
	securityParam := uint8(4)
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(
		context.Background(),
		prover.TreeConfig{Datadir: t.TempDir()},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		time.Now().Add(duration),
		securityParam,
	)
	r.NoError(err)
	err = Validate(*merkleProof, BadLabelHashFunc, hash.GenMerkleHashFunc(challenge), leafs, securityParam)
	r.Error(err)
	r.Regexp("label at index 0 incorrect - expected: [0-f]* actual: [0-f]*", err.Error())
}
