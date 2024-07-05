package registration

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

func genChallenge() ([]byte, error) {
	challenge := make([]byte, 32)
	_, err := rand.Read(challenge)
	return challenge, err
}

func genChallenges(num int) ([][]byte, error) {
	ch := make([][]byte, num)
	for i := 0; i < num; i++ {
		challenge, err := genChallenge()
		if err != nil {
			return nil, err
		}
		ch[i] = challenge
	}

	return ch, nil
}

func newTestRound(t *testing.T, epoch uint, opts ...newRoundOptionFunc) *round {
	t.Helper()
	r, err := newRound(epoch, t.TempDir(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })
	return r
}

// Test submitting many challenges.
func TestRound_Submit(t *testing.T) {
	t.Parallel()
	t.Run("submit many different challenges", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t, 0)
		challenges, err := genChallenges(32)
		require.NoError(t, err)

		// Act
		var done <-chan error
		for _, ch := range challenges {
			done, err = round.submit(context.Background(), ch, ch)
			require.NoError(t, err)
		}
		require.NoError(t, <-done)

		// Verify
		require.ElementsMatch(t, challenges, round.getMembers())
	})
	t.Run("submit challenges with same key (submits flushed)", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t, 0)
		challenges, err := genChallenges(2)
		require.NoError(t, err)

		// Act
		_, err = round.submit(context.Background(), []byte("key"), challenges[0])
		require.NoError(t, err)
		round.flushPendingSubmits()

		_, err = round.submit(context.Background(), []byte("key"), challenges[0])
		require.ErrorIs(t, err, ErrChallengeAlreadySubmitted)

		_, err = round.submit(context.Background(), []byte("key"), challenges[1])
		require.ErrorIs(t, err, ErrConflictingRegistration)

		// Verify
		require.ElementsMatch(t, [][]byte{challenges[0]}, round.getMembers())
		challenge, err := round.db.Get([]byte("key"), nil)
		require.NoError(t, err)
		require.Equal(t, challenges[0], challenge)
	})
	t.Run("submit challenges with same key (detect pending)", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t, 0, withSubmitFlushInterval(time.Hour))
		challenges, err := genChallenges(2)
		require.NoError(t, err)

		// Act
		_, err = round.submit(context.Background(), []byte("key"), challenges[0])
		require.NoError(t, err)

		_, err = round.submit(context.Background(), []byte("key"), challenges[0])
		require.ErrorIs(t, err, ErrChallengeAlreadySubmitted)

		_, err = round.submit(context.Background(), []byte("key"), challenges[1])
		require.ErrorIs(t, err, ErrConflictingRegistration)

		// Verify
		round.flushPendingSubmits()
		require.ElementsMatch(t, [][]byte{challenges[0]}, round.getMembers())
		challenge, err := round.db.Get([]byte("key"), nil)
		require.NoError(t, err)
		require.Equal(t, challenges[0], challenge)
	})
	t.Run("cannot oversubmit", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t, 0, withMaxMembers(2))
		challenges, err := genChallenges(3)
		require.NoError(t, err)

		// Act
		_, err = round.submit(context.Background(), challenges[0], challenges[0])
		require.NoError(t, err)
		_, err = round.submit(context.Background(), challenges[1], challenges[1])
		require.NoError(t, err)
		_, err = round.submit(context.Background(), challenges[2], challenges[2])
		require.ErrorIs(t, err, ErrMaxMembersReached)

		round.flushPendingSubmits()

		// Verify
		require.ElementsMatch(t, [][]byte{challenges[0], challenges[1]}, round.getMembers())
		for _, ch := range challenges[:2] {
			challengeInDb, err := round.db.Get(ch, nil)
			require.NoError(t, err)
			require.Equal(t, ch, challengeInDb)
		}
	})
}

func TestRound_Reopen(t *testing.T) {
	t.Parallel()
	dbdir := t.TempDir()
	// Arrange
	challenge, err := genChallenge()
	require.NoError(t, err)
	{
		round, err := newRound(0, dbdir)
		require.NoError(t, err)
		_, err = round.submit(context.Background(), []byte("key"), challenge)
		require.NoError(t, err)
		require.NoError(t, round.Close())
	}

	// Act
	recovered, err := newRound(0, dbdir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, recovered.Close()) })

	// Verify
	require.Equal(t, [][]byte{challenge}, recovered.getMembers())
	require.Equal(t, 1, recovered.members)
}

func TestRound_FlushingPending(t *testing.T) {
	t.Parallel()
	t.Run("timed flush detects it was canceled", func(t *testing.T) {
		t.Parallel()
		round := newTestRound(t, 0, withSubmitFlushInterval(time.Hour))
		// submit schedules a flush in 1 hour
		round.submit(context.Background(), []byte("key"), []byte("challenge"))
		// cancel the flush
		round.pendingFlush.Stop()
		round.pendingFlush = nil

		// simulate a flush called by the timer
		round.timedFlushPendingSubmits()
		// flush should not have happened
		_, err := round.db.Get([]byte("key"), nil)
		require.ErrorIs(t, err, leveldb.ErrNotFound)
	})
	t.Run("canceling timed flush", func(t *testing.T) {
		t.Parallel()
		round := newTestRound(t, 0, withSubmitFlushInterval(time.Hour))
		// submit schedules a flush in 1 hour
		round.submit(context.Background(), []byte("key"), []byte("challenge"))
		// manual flush cancels the timer
		round.flushPendingSubmits()
		// timer should be canceled
		require.Nil(t, round.pendingFlush)
	})
}
