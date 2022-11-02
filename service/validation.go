package service

import (
	"bytes"
	"context"
	"fmt"

	postShared "github.com/spacemeshos/post/shared"

	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/types"
)

// validateChallenge verifies if the contents of the challenge are valid.
// It checks:
// - initial post metadata against PostConfig
// - initial post proof using the provided proof verifier function
// - if previous ATX's node ID matches the node ID in the challenge.
func validateChallenge(ctx context.Context, challenge *shared.Challenge, atxs types.AtxProvider, postConfig *types.PostConfig, proofVerifier func(*postShared.Proof, *postShared.ProofMetadata) error) error {
	if initialPost := challenge.InitialPost; initialPost != nil {
		meta := &initialPost.Metadata
		if meta.BitsPerLabel != postConfig.BitsPerLabel {
			return fmt.Errorf("validation: BitsPerLabel mismatch: challenge: %d, config: %d", meta.BitsPerLabel, postConfig.BitsPerLabel)
		}
		if meta.LabelsPerUnit != postConfig.LabelsPerUnit {
			return fmt.Errorf("validation: LabelsPerUnit mismatch: challenge: %d, config: %d", meta.LabelsPerUnit, postConfig.LabelsPerUnit)
		}
		if meta.K1 != postConfig.K1 {
			return fmt.Errorf("validation: K1 mismatch: challenge: %d, config: %d", meta.K1, postConfig.K1)
		}
		if meta.K2 != postConfig.K2 {
			return fmt.Errorf("validation: K2 mismatch: challenge: %d, config: %d", meta.K2, postConfig.K2)
		}
		if meta.NumUnits < postConfig.MinNumUnits || meta.NumUnits > postConfig.MaxNumUnits {
			return fmt.Errorf("validation: NumUnits not in the required range [%d <= %d <= %d]", postConfig.MinNumUnits, meta.NumUnits, postConfig.MaxNumUnits)
		}
		return proofVerifier(&initialPost.Proof, &initialPost.Metadata)
	}
	// Verify against previous ATX
	{
		atx, err := atxs.Get(ctx, challenge.PreviousATXId)
		if err != nil {
			return fmt.Errorf("validation: failed to fetch previous ATX (%w)", err)
		}
		if !bytes.Equal(challenge.NodeID, atx.NodeID) {
			return fmt.Errorf("validation: NodeID mismtach: challenge: %X, previous ATX: %X", challenge.NodeID, atx.NodeID)
		}
		if challenge.PubLayerId != atx.PubLayerID+1 {
			return fmt.Errorf("validation: Publayer ID mismatch: challenge publayer ID: %d, previous ATX publayer ID: %d", challenge.PubLayerId, atx.PubLayerID)
		}
	}
	// Verify against positioning ATX
	{
		atx, err := atxs.Get(ctx, challenge.PositioningAtxId)
		if err != nil {
			return fmt.Errorf("validation: failed to fetch positioning ATX (%w)", err)
		}
		if challenge.PubLayerId != atx.PubLayerID+1 {
			return fmt.Errorf("validation: Publayer ID mismatch: challenge publayer ID: %d, positioning ATX publayer ID: %d", challenge.PubLayerId, atx.PubLayerID)
		}
	}
	return nil
}
