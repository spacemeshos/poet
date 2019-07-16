package service

import (
	"bytes"
	"crypto/rand"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type MockBroadcaster struct {
	receivedMessages chan []byte
}

func (b *MockBroadcaster) BroadcastProof(msg []byte) error {
	b.receivedMessages <- msg
	return nil
}

func TestNewService(t *testing.T) {
	req := require.New(t)

	cfg := new(Config)
	cfg.N = 17
	cfg.InitialRoundDuration = 1 * time.Second

	s, err := NewService(cfg)
	req.NoError(err)

	proofBroadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	s.Start(proofBroadcaster)

	type challenge struct {
		data  []byte
		round *round
	}

	challengesCount := 8
	challenges := make([]challenge, challengesCount)
	info := s.Info()

	// Generate random challenges.
	for i := 0; i < len(challenges); i++ {
		challenges[i] = challenge{data: make([]byte, 32)}
		_, err := rand.Read(challenges[i].data)
		req.NoError(err)
	}

	// Submit challenges.
	for i := 0; i < len(challenges); i++ {
		round, err := s.Submit(challenges[i].data)
		req.NoError(err)
		req.Equal(info.OpenRoundId, round.Id)
		challenges[i].round = round

		// Verify that all submissions returned the same round instance.
		if i > 0 {
			req.Equal(challenges[i].round, challenges[i-1].round)
		}
	}

	// Verify that round is still open.
	req.Equal(info.OpenRoundId, s.Info().OpenRoundId)

	// Wait for round closure.
	select {
	case <-challenges[0].round.closedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Verify that round iteration proceeded.
	prevInfo := info
	info = s.Info()
	req.Equal(prevInfo.OpenRoundId+1, info.OpenRoundId)
	req.Contains(info.ExecutingRoundsIds, prevInfo.OpenRoundId)

	// Wait for execution completion.
	select {
	case <-challenges[0].round.executedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Wait for proof message broadcast.
	select {
	case msg := <-proofBroadcaster.receivedMessages:
		poetProof := PoetProofMessage{}
		_, err := xdr.Unmarshal(bytes.NewReader(msg), &poetProof)
		req.NoError(err)
	case <-time.After(100 * time.Millisecond):
		req.Fail("proof message wasn't sent")
	}
}
