package service

import (
	"crypto/rand"
	"github.com/spacemeshos/merkle-tree"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	req := require.New(t)

	cfg := new(Config)
	cfg.N = 9
	cfg.HashFunction = "sha256"
	cfg.InitialRoundDuration = 1 * time.Second

	s, err := NewService(cfg)
	req.NoError(err)

	var lastRoundId uint
	var lastCommitment []byte
	size := 64
	for i := 0; i < size; i++ {
		x := make([]byte, 32)
		_, err := rand.Read(x)
		req.NoError(err)

		res, err := s.SubmitCommitment(x)
		req.NoError(err)

		lastRoundId = res.roundId
		lastCommitment = x
	}

	round, err := s.Round(lastRoundId)
	req.NoError(err)
	req.Equal(size, len(round.commitments))

	<-round.closedChan
	//t.Logf("round merkle tree root: %v", round.treeRoot)

	<-round.executedChan
	//t.Logf("round merkle tree root phi: %v", round.phi)

	proof, err := s.MembershipProof(lastRoundId, lastCommitment)
	req.NoError(err)
	//t.Logf("commitment %v proof: %v", lastCommitment, proof)

	leafIndices := []uint64{uint64(size - 1)}
	leaves := [][]byte{lastCommitment}
	valid, err := merkle.ValidatePartialTree(leafIndices, leaves, proof, round.treeRoot, merkle.GetSha256Parent)
	req.NoError(err)
	req.True(valid)
}
