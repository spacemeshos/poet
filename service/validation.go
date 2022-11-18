package service

import (
	"bytes"
	"context"
	"fmt"

	postShared "github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signing"
	"github.com/spacemeshos/poet/types"
)

type challengeValidator struct {
	postConfig     *types.PostConfig
	atxs           types.AtxProvider
	proofVerifier  func(*postShared.Proof, *postShared.ProofMetadata) error
	layersPerEpoch uint
}

// validateChallenge verifies if the contents of the challenge are valid.
// It checks:
// - initial post metadata against PostConfig
// - initial post proof using the provided proof verifier function
// - if previous ATX's node ID matches the node ID in the challenge.
func (v *challengeValidator) validateChallenge(ctx context.Context, signedChallenge signing.Signed[shared.Challenge]) error {
	logger := logging.FromContext(ctx)
	challenge := signedChallenge.Data()
	nodeID := signedChallenge.PubKey()
	if initialPost := challenge.InitialPost; initialPost != nil {
		meta := &initialPost.Metadata
		if meta.BitsPerLabel != v.postConfig.BitsPerLabel {
			return fmt.Errorf("validation: BitsPerLabel mismatch: challenge: %d, config: %d", meta.BitsPerLabel, v.postConfig.BitsPerLabel)
		}
		if meta.LabelsPerUnit != v.postConfig.LabelsPerUnit {
			return fmt.Errorf("validation: LabelsPerUnit mismatch: challenge: %d, config: %d", meta.LabelsPerUnit, v.postConfig.LabelsPerUnit)
		}
		if meta.K1 != v.postConfig.K1 {
			return fmt.Errorf("validation: K1 mismatch: challenge: %d, config: %d", meta.K1, v.postConfig.K1)
		}
		if meta.K2 != v.postConfig.K2 {
			return fmt.Errorf("validation: K2 mismatch: challenge: %d, config: %d", meta.K2, v.postConfig.K2)
		}
		if meta.NumUnits < v.postConfig.MinNumUnits || meta.NumUnits > v.postConfig.MaxNumUnits {
			return fmt.Errorf("validation: NumUnits not in the required range [%d <= %d <= %d]", v.postConfig.MinNumUnits, meta.NumUnits, v.postConfig.MaxNumUnits)
		}
		return v.proofVerifier(&initialPost.Proof, &initialPost.Metadata)
	}
	// Verify against previous ATX
	{
		if challenge.PreviousATXId == nil {
			return fmt.Errorf("missing previous ATX ID")
		}
		atx, err := v.atxs.Get(ctx, *challenge.PreviousATXId)
		if err != nil {
			return fmt.Errorf("validation: failed to fetch previous ATX (%w)", err)
		}
		logger.With().Debug("got previous ATX", log.Field(zap.Object("atx", atx)))
		if !bytes.Equal(nodeID, atx.NodeID) {
			return fmt.Errorf("validation: NodeID mismatch: challenge: %X, previous ATX: %X", nodeID, atx.NodeID)
		}
		if !(uint(challenge.PubLayerId) >= uint(atx.PubLayerID)+v.layersPerEpoch) {
			return fmt.Errorf("validation: Publayer ID mismatch: challenge publayer ID: %d, previous ATX publayer ID: %d", challenge.PubLayerId, atx.PubLayerID)
		}
	}
	// Verify against positioning ATX
	{
		atx, err := v.atxs.Get(ctx, challenge.PositioningAtxId)
		if err != nil {
			return fmt.Errorf("validation: failed to fetch positioning ATX (%w)", err)
		}
		logger.With().Debug("got positioning ATX", log.Field(zap.Object("atx", atx)))
		if !(uint(challenge.PubLayerId) >= uint(atx.PubLayerID)+v.layersPerEpoch) {
			return fmt.Errorf("validation: Publayer ID mismatch: challenge publayer ID: %d, positioning ATX publayer ID: %d", challenge.PubLayerId, atx.PubLayerID)
		}
	}
	return nil
}
