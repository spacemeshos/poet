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
	cfg.N = 17
	cfg.HashFunction = "sha256"
	cfg.InitialRoundDuration = 1 * time.Second

	s, err := NewService(cfg)
	req.NoError(err)

	type commit struct {
		data  []byte
		round *round
	}

	numOfCommits := 8
	commits := make([]commit, numOfCommits)
	info := s.Info()

	// Generate random commits.
	for i := 0; i < len(commits); i++ {
		commits[i] = commit{data: make([]byte, 32)}
		_, err := rand.Read(commits[i].data)
		req.NoError(err)
	}

	// Submit commits.
	for i := 0; i < len(commits); i++ {
		round, err := s.Submit(commits[i].data)
		req.NoError(err)
		req.Equal(info.OpenRoundId, round.Id)
		commits[i].round = round

		// Verify that all submissions returned the same round instance.
		if i > 0 {
			req.Equal(commits[i].round, commits[i-1].round)
		}
	}

	// Verify round info.
	roundInfo, err := s.RoundInfo(info.OpenRoundId)
	req.NoError(err)
	req.Equal(numOfCommits, roundInfo.NumOfCommits)

	// Verify that found is still open.
	req.Equal(info.OpenRoundId, s.Info().OpenRoundId)

	// Wait for round closure.
	select {
	case <-commits[0].round.closedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Verify that round iteration proceeded.
	prevInfo := info
	info = s.Info()
	req.Equal(prevInfo.OpenRoundId+1, info.OpenRoundId)
	req.Contains(info.ExecutingRoundsIds, prevInfo.OpenRoundId)
	req.NotContains(info.ExecutedRoundsIds, prevInfo.OpenRoundId)

	// Verify the membership proof of each commit.
	for i, commit := range commits {
		proof, err := s.MembershipProof(commit.round.Id, commit.data, false)
		req.NoError(err)

		leafIndices := []uint64{uint64(i)}
		leaves := [][]byte{commit.data}
		valid, err := merkle.ValidatePartialTree(leafIndices, leaves, proof, commit.round.merkleRoot, merkle.GetSha256Parent)
		req.NoError(err)
		req.True(valid)
	}
}
