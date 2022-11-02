package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	postShared "github.com/spacemeshos/post/shared"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/types"
	"github.com/spacemeshos/poet/types/mock_types"
)

var errInvalidPostProof = errors.New("invalid post proof")

func acceptProof(*postShared.Proof, *postShared.ProofMetadata) error { return nil }
func denyProof(*postShared.Proof, *postShared.ProofMetadata) error   { return errInvalidPostProof }

func createChallengeWithInitialPost(postConfig *types.PostConfig, overrides ...func(*shared.Challenge)) *shared.Challenge {
	challenge := shared.Challenge{
		PositioningAtxId: []byte("positioning-atx"),
		PubLayerId:       0,
		InitialPost: &shared.InitialPost{
			Metadata: postShared.ProofMetadata{
				NumUnits:      5,
				BitsPerLabel:  postConfig.BitsPerLabel,
				LabelsPerUnit: postConfig.LabelsPerUnit,
				K1:            postConfig.K1,
				K2:            postConfig.K2,
			},
		},
	}

	for _, o := range overrides {
		o(&challenge)
	}
	return &challenge
}

// Test challenge validation for challenges carrying an initial Post challenge.
func Test_ChallengeValidation_Initial(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	postConfig := types.PostConfig{
		MinNumUnits:   1,
		MaxNumUnits:   10,
		BitsPerLabel:  4,
		LabelsPerUnit: 8,
		K1:            2,
		K2:            6,
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		require.NoError(validateChallenge(context.Background(), createChallengeWithInitialPost(&postConfig), mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof))
	})
	t.Run("invalid post proof", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig)

		require.ErrorIs(
			validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, denyProof),
			errInvalidPostProof,
		)
	})
	t.Run("invalid post metadata (NumUnits < MinNumUnits)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.NumUnits = 0 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "NumUnits")
	})
	t.Run("invalid post metadata (NumUnits > MaxNumUnits)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.NumUnits = postConfig.MaxNumUnits + 1 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "NumUnits")
	})
	t.Run("invalid post metadata (BitsPerLabel)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.BitsPerLabel = postConfig.BitsPerLabel + 1 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "BitsPerLabel")
	})
	t.Run("invalid post metadata (LabelsPerUnit)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.LabelsPerUnit = postConfig.LabelsPerUnit + 1 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "LabelsPerUnit")
	})
	t.Run("invalid post metadata (K1)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.K1 = postConfig.K1 + 1 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "K1")
	})
	t.Run("invalid post metadata (K2)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := createChallengeWithInitialPost(&postConfig, func(c *shared.Challenge) { c.InitialPost.Metadata.K2 = postConfig.K2 + 1 })
		require.ErrorContains(validateChallenge(context.Background(), challenge, mock_types.NewMockAtxProvider(ctrl), &postConfig, acceptProof), "K2")
	})
}

// Test challenge validation for subsequent challenges that
// use previous ATX as the part of the challenge.
func Test_ChallengeValidation_NonInitial(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	nodeId := []byte("nodeid")
	posAtxnodeId := []byte("positioning-atx-nodeid")
	prevAtxId := shared.ATXID("previousATX")
	posAtxId := shared.ATXID("positioningATX")

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(&types.ATX{NodeID: nodeId, PubLayerID: 9}, nil)
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(&types.ATX{NodeID: posAtxnodeId, PubLayerID: 9}, nil)

		require.NoError(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof))
	})
	t.Run("NodeID doesn't match previous ATX NodeID", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(&types.ATX{NodeID: []byte("other"), PubLayerID: 9}, nil)
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(&types.ATX{NodeID: posAtxnodeId, PubLayerID: 9}, nil)

		require.ErrorContains(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof), "NodeID")
	})
	t.Run("previous ATX unavailable", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(nil, errors.New("fail"))
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(&types.ATX{NodeID: posAtxnodeId, PubLayerID: 9}, nil)

		require.ErrorContains(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof), "failed to fetch")
	})
	t.Run("positioning ATX unavailable", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(&types.ATX{NodeID: nodeId, PubLayerID: 9}, nil)
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(nil, errors.New("fail"))

		require.ErrorContains(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof), "failed to fetch")
	})
	t.Run("publayerID != previousATX.publayerID + 1", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(&types.ATX{NodeID: nodeId, PubLayerID: 0}, nil)
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(&types.ATX{NodeID: posAtxnodeId, PubLayerID: 9}, nil)

		require.ErrorContains(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof), "Publayer ID mismatch")
	})
	t.Run("publayerID != positioningATX.publayerID + 1", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		challenge := shared.Challenge{
			PositioningAtxId: posAtxId,
			PubLayerId:       10,
			PreviousATXId:    prevAtxId,
			NodeID:           nodeId,
		}

		atxs := mock_types.NewMockAtxProvider(ctrl)
		atxs.EXPECT().Get(gomock.Any(), prevAtxId).AnyTimes().Return(&types.ATX{NodeID: nodeId, PubLayerID: 9}, nil)
		atxs.EXPECT().Get(gomock.Any(), posAtxId).AnyTimes().Return(&types.ATX{NodeID: posAtxnodeId, PubLayerID: 0}, nil)

		require.ErrorContains(validateChallenge(context.Background(), &challenge, atxs, &types.PostConfig{}, acceptProof), "Publayer ID mismatch")
	})
}
