package verifier

import (
	"bytes"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/poet/shared"
	"sort"
)

func Validate(proof shared.MerkleProof, challenge shared.NewChallenge, leafCount uint64, securityParam uint8) error {
	if int(securityParam) != len(proof.ProvenLeaves) {
		return fmt.Errorf("number of proven leaves (%d) must be equal to security param (%d)",
			len(proof.ProvenLeaves), securityParam)
	}
	provenLeafIndices := asSortedSlice(shared.FiatShamir(proof.Root, leafCount, securityParam))
	valid, parkingSnapshots, err := merkle.ValidatePartialTreeWithParkingSnapshots(provenLeafIndices,
		proof.ProvenLeaves, proof.ProofNodes, proof.Root, challenge.MerkleHashFunc())
	if err != nil {
		return fmt.Errorf("error while validating merkle proof: %v", err)
	}
	if !valid {
		return fmt.Errorf("merkle proof not valid")
	}

	for id, label := range proof.ProvenLeaves {
		expectedLabel := shared.MakeLabel(challenge.LabelHashFunc(), provenLeafIndices[id], parkingSnapshots[id])
		if !bytes.Equal(expectedLabel, label) {
			return fmt.Errorf("label at index %d incorrect", id)
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
