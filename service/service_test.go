package service

import (
	"crypto/rand"
	"github.com/spacemeshos/merkle-tree"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO(moshababo): add tests

func TestNewService(t *testing.T) {
	req := require.New(t)

	cfg := new(Config)
	cfg.N = 9
	cfg.HashFunction = "sha256"
	cfg.InitialRoundDuration = 1 * time.Second

	s, err := NewService(cfg)
	req.NoError(err)

	var lastRoundId int
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

	info, err := s.RoundInfo(lastRoundId)
	req.NoError(err)
	req.Equal(size, info.numOfCommitments)

	round, err := s.round(lastRoundId)
	req.NoError(err)
	req.Equal(size, info.numOfCommitments)

	<-round.closedChan
	//t.Logf("round merkle merkleTree root: %v", round.merkleRoot)

	<-round.executedChan
	//t.Logf("round merkle merkleTree root phi: %v", round.phi)

	proof, err := s.MembershipProof(lastRoundId, lastCommitment)
	req.NoError(err)
	//t.Logf("commitment %v proof: %v", lastCommitment, proof)

	leafIndices := []uint64{uint64(size - 1)}
	leaves := [][]byte{lastCommitment}
	valid, err := merkle.ValidatePartialTree(leafIndices, leaves, proof, round.merkleRoot, merkle.GetSha256Parent)
	req.NoError(err)
	req.True(valid)
}

//func TestService_Info(t *testing.T) {
//	req := require.New(t)
//
//	cfg := new(Config)
//	cfg.N = 15
//	cfg.HashFunction = "sha256"
//	cfg.InitialRoundDuration = 1 * time.Second
//	cfg.ExecuteEmpty = true
//
//	s, err := NewService(cfg)
//	req.NoError(err)
//
//
//	size := 64
//	for i := 0; i < size; i++ {
//		x := make([]byte, 32)
//		_, err := rand.Read(x)
//		req.NoError(err)
//
//		_, err = s.SubmitCommitment(x)
//		req.NoError(err)
//	}
//
//	t.Logf("Server info: %+v", s.Info())
//
//	<-time.After(2 * time.Second)
//	t.Logf("Server info: %+v", s.Info())
//
//	<-time.After(1 * time.Second)
//	t.Logf("Server info: %+v", s.Info())
//
//	roundInfo, _ := s.RoundInfo(1)
//	t.Logf("Round info: %+v", roundInfo)
//}
