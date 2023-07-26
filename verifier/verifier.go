package verifier

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/spacemeshos/merkle-tree"

	"github.com/spacemeshos/poet/shared"
)

// Validate verifies that a Merkle proof was generated by an honest PoET prover. It validates that the number of proven
// leaves matches the security param, validates the Merkle proof itself and verifies the labels are derived from the
// left cousins in the Merkle tree.
func Validate(proof shared.MerkleProof, labelHashFunc func(data []byte) []byte,
	merkleHashFunc merkle.HashFunc, numLeaves uint64, securityParam uint8,
) error {
	if int(securityParam) != len(proof.ProvenLeaves) {
		return fmt.Errorf("number of proven leaves (%d) must be equal to security param (%d)",
			len(proof.ProvenLeaves), securityParam)
	}
	if len(proof.ProvenLeaves)*36 < len(proof.ProofNodes) {
		return fmt.Errorf("for every proven leaf (%d) there must be at most 36 proof nodes (%d)",
			len(proof.ProvenLeaves), len(proof.ProofNodes),
		)
	}
	if len(proof.ProvenLeaves) > len(proof.ProofNodes) {
		return fmt.Errorf("for every proven leaf (%d) there must be at least 1 proof node (%d)",
			len(proof.ProvenLeaves), len(proof.ProofNodes),
		)
	}

	provenLeafIndices := asSortedSlice(shared.FiatShamir(proof.Root, numLeaves, securityParam))
	provenLeaves := make([][]byte, 0, len(proof.ProvenLeaves))
	for i := range proof.ProvenLeaves {
		provenLeaves = append(provenLeaves, proof.ProvenLeaves[i][:])
	}
	proofNodes := make([][]byte, 0, len(proof.ProofNodes))
	for i := range proof.ProofNodes {
		proofNodes = append(proofNodes, proof.ProofNodes[i][:])
	}
	valid, parkingSnapshots, err := merkle.ValidatePartialTreeWithParkingSnapshots(
		provenLeafIndices,
		provenLeaves,
		proofNodes,
		proof.Root,
		merkleHashFunc,
	)
	if err != nil {
		return fmt.Errorf("error while validating merkle proof: %v", err)
	}
	if !valid {
		return fmt.Errorf("merkle proof not valid")
	}

	if len(parkingSnapshots) != len(proof.ProvenLeaves) {
		return fmt.Errorf("merkle proof incomplete")
	}
	makeLabel := shared.MakeLabelFunc()
	for id, label := range proof.ProvenLeaves {
		expectedLabel := makeLabel(labelHashFunc, provenLeafIndices[id], parkingSnapshots[id])
		if !bytes.Equal(expectedLabel, label[:]) {
			return fmt.Errorf("label at index %d incorrect - expected: %x actual: %x", id, expectedLabel, label)
		}
	}

	return nil
}

func asSortedSlice(s map[uint64]bool) []uint64 {
	var ret []uint64
	for key, value := range s {
		if value {
			ret = append(ret, key)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}
