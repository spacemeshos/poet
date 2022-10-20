package service

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/merkle-tree"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/signal"
)

type MockBroadcaster struct {
	receivedMessages chan []byte
}

func (b *MockBroadcaster) BroadcastProof(msg []byte, roundID string, members [][]byte) error {
	b.receivedMessages <- msg
	return nil
}

type challenge struct {
	data  []byte
	round *round
}

func TestService_Recovery(t *testing.T) {
	req := require.New(t)
	sig := signal.NewSignal()
	broadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	cfg := &Config{
		Genesis:       time.Now().Add(time.Second).Format(time.RFC3339),
		EpochDuration: time.Second,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}

	tempdir := t.TempDir()
	// Create a new service instance.
	s, err := NewService(sig, cfg, tempdir)
	req.NoError(err)
	err = s.Start(broadcaster)
	req.NoError(err)

	// Track the service rounds.
	numRounds := 3
	rounds := make([]*round, numRounds)

	// Generate 4 groups of random challenges.
	challengeGroupSize := 40
	challengeGroups := make([][]challenge, 4)
	for i := 0; i < 4; i++ {
		challengeGroup := make([]challenge, challengeGroupSize)
		for i := 0; i < challengeGroupSize; i++ {
			challengeGroup[i] = challenge{data: make([]byte, 32)}
			_, err := rand.Read(challengeGroup[i].data)
			req.NoError(err)
		}
		challengeGroups[i] = challengeGroup
	}

	submittedChallenges := make(map[int][]challenge)
	submitChallenges := func(roundIndex int, groupIndex int) {
		challengesGroup := challengeGroups[groupIndex]
		for i := 0; i < len(challengeGroups[groupIndex]); i++ {
			round, err := s.Submit(challengesGroup[i].data)
			req.NoError(err)
			req.Equal(strconv.Itoa(roundIndex), round.ID)

			// Verify that all submissions returned the same round instance.
			if rounds[roundIndex] == nil {
				rounds[roundIndex] = round
			} else {
				req.Equal(rounds[roundIndex], round)
			}
		}

		// Track the submitted challenges per-round for later validation.
		submittedChallenges[roundIndex] = append(submittedChallenges[roundIndex], challengesGroup...)
	}

	// Submit challenges to open round (0).
	submitChallenges(0, 0)

	// Verify that round is still open.
	req.Equal(rounds[0].ID, s.openRoundID())

	// Wait for round 0 to start executing.
	select {
	case <-rounds[0].executionStartedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Verify that round iteration proceeds: a new round opened, previous round is executing.
	req.Contains(s.executingRounds, rounds[0].ID)
	s.openRoundMutex.Lock()
	rounds[1] = s.openRound
	s.openRoundMutex.Unlock()

	// Submit challenges to open round (1).
	submitChallenges(1, 1)

	// Request shutdown.
	sig.RequestShutdown()

	// Verify shutdown error is received.
	select {
	case err := <-s.errChan:
		req.EqualError(err, fmt.Sprintf("round %v execution error: %v", rounds[0].ID, prover.ErrShutdownRequested.Error()))
	case <-rounds[0].executionEndedChan:
		req.Fail("round execution ended instead of shutting down")
	}

	// Verify service state. should have no open or executing rounds. Check after a delay to allow service to update state.
	time.Sleep(100 * time.Millisecond)
	s.openRoundMutex.Lock()
	req.Nil(s.openRound)
	s.openRoundMutex.Unlock()
	req.Equal(len(s.executingRounds), 0)

	// Create a new service instance.
	sig = signal.NewSignal()
	s, err = NewService(sig, cfg, tempdir)
	req.NoError(err)

	err = s.Start(broadcaster)
	req.NoError(err)

	// Service instance should recover 2 rounds: round 1 in executing state, and round 2 in open state.
	req.Equal(1, len(s.executingRounds))
	first, ok := s.executingRounds["0"]
	req.True(ok)
	req.Equal(s.openRoundID(), "1")
	rounds[0] = first
	s.openRoundMutex.Lock()
	rounds[1] = s.openRound
	s.openRoundMutex.Unlock()

	select {
	case <-rounds[1].executionStartedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	submitChallenges(2, 2)

	select {
	case <-rounds[2].executionEndedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Verify that new service instance broadcast 3 distinct rounds proofs, by the expected order.
	for i := 0; i < numRounds; i++ {
		proofMsg := PoetProofMessage{}
		select {
		case <-time.After(10 * time.Second):
			req.Fail("proof message wasn't sent")
		case msg := <-broadcaster.receivedMessages:
			dec := scale.NewDecoder(bytes.NewReader(msg))
			_, err := proofMsg.DecodeScale(dec)
			req.NoError(err)
		}

		// Verify the submitted challenges.
		req.Len(proofMsg.Members, len(submittedChallenges[i]))
		for _, ch := range submittedChallenges[i] {
			req.True(contains(proofMsg.Members, ch.data), "proof %v, round %v", proofMsg.RoundID, i)
		}

		// Verify round statement.
		mtree, err := merkle.NewTree()
		req.NoError(err)
		for _, m := range proofMsg.Members {
			req.NoError(mtree.AddLeaf(m))
		}
		proof, err := rounds[i].proof(false)
		req.NoError(err, "round %d", i)
		req.Equal(mtree.Root(), proof.Statement)
	}

	// Request shutdown.
	sig.RequestShutdown()
	time.Sleep(100 * time.Millisecond)
}

func contains(list [][]byte, item []byte) bool {
	for _, listItem := range list {
		if bytes.Equal(listItem, item) {
			return true
		}
	}

	return false
}

func TestNewService(t *testing.T) {
	req := require.New(t)
	tempdir := t.TempDir()

	cfg := new(Config)
	cfg.Genesis = time.Now().Add(time.Second / 2).Format(time.RFC3339)
	cfg.EpochDuration = time.Second
	cfg.PhaseShift = time.Second / 2
	cfg.CycleGap = time.Second / 4

	sig := signal.NewSignal()
	s, err := NewService(sig, cfg, tempdir)
	req.NoError(err)
	proofBroadcaster := &MockBroadcaster{receivedMessages: make(chan []byte)}
	err = s.Start(proofBroadcaster)
	req.NoError(err)

	challengesCount := 8
	challenges := make([]challenge, challengesCount)

	// Generate random challenges.
	for i := 0; i < len(challenges); i++ {
		challenges[i] = challenge{data: make([]byte, 32)}
		_, err := rand.Read(challenges[i].data)
		req.NoError(err)
	}

	info, err := s.Info()
	require.NoError(t, err)
	currentRound := info.OpenRoundID

	// Submit challenges.
	for i := 0; i < len(challenges); i++ {
		round, err := s.Submit(challenges[i].data)
		req.NoError(err)
		req.Equal(currentRound, round.ID)
		challenges[i].round = round

		// Verify that all submissions returned the same round instance.
		if i > 0 {
			req.Equal(challenges[i].round, challenges[i-1].round)
		}
	}

	// Verify that round is still open.
	info, err = s.Info()
	req.NoError(err)
	req.Equal(currentRound, info.OpenRoundID)

	// Wait for round to start execution.
	select {
	case <-challenges[0].round.executionStartedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Verify that round iteration proceeded.
	prevInfo := info
	info, err = s.Info()
	req.NoError(err)
	prevIndex, err := strconv.Atoi(prevInfo.OpenRoundID)
	req.NoError(err)
	req.Equal(fmt.Sprintf("%d", prevIndex+1), info.OpenRoundID)
	req.Contains(info.ExecutingRoundsIds, prevInfo.OpenRoundID)

	// Wait for end of execution.
	select {
	case <-challenges[0].round.executionEndedChan:
	case err := <-s.errChan:
		req.Fail(err.Error())
	}

	// Wait for proof message broadcast.
	select {
	case msg := <-proofBroadcaster.receivedMessages:
		poetProof := PoetProofMessage{}
		dec := scale.NewDecoder(bytes.NewReader(msg))
		_, err := poetProof.DecodeScale(dec)
		req.NoError(err)
	case <-time.After(100 * time.Millisecond):
		req.Fail("proof message wasn't sent")
	}

	// Request shutdown.
	sig.RequestShutdown()
	time.Sleep(100 * time.Millisecond)
}

func genChallenges(num int) ([][]byte, error) {
	ch := make([][]byte, num)
	for i := 0; i < num; i++ {
		ch[i] = make([]byte, 32)
		_, err := rand.Read(ch[i])
		if err != nil {
			return nil, err
		}
	}

	return ch, nil
}
