package service

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
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

func numChallenges(r *round) int {
	iter := r.challengesDb.NewIterator(nil, nil)
	defer iter.Release()

	var num int
	for iter.Next() {
		num++
	}

	return num
}

type newRoundOption struct {
	epoch      uint32
	maxMembers uint
}

type newRoundOptionFunc func(*newRoundOption)

func withMaxMembers(maxMembers uint) newRoundOptionFunc {
	return func(o *newRoundOption) {
		o.maxMembers = maxMembers
	}
}

func newTestRound(t *testing.T, opts ...newRoundOptionFunc) *round {
	t.Helper()
	options := newRoundOption{
		epoch:      7,
		maxMembers: 10,
	}
	for _, opt := range opts {
		opt(&options)
	}

	round, err := newRound(t.TempDir(), options.epoch, options.maxMembers)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, round.teardown(true)) })
	return round
}

// validateProof validates proof from round's execution state.
func validateProof(t *testing.T, execution *executionState) {
	t.Helper()
	req := require.New(t)
	req.NotNil(execution.NIP)
	req.NoError(
		verifier.Validate(
			*execution.NIP,
			hash.GenLabelHashFunc(execution.Statement),
			hash.GenMerkleHashFunc(execution.Statement),
			execution.NumLeaves,
			shared.T,
		),
	)
}

func TestRound_TearDown(t *testing.T) {
	t.Run("no cleanup", func(t *testing.T) {
		// Arrange
		round, err := newRound(t.TempDir(), 0, 1)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.teardown(false))

		// Verify
		_, err = os.Stat(round.datadir)
		require.NoError(t, err)
	})

	t.Run("cleanup", func(t *testing.T) {
		// Arrange
		round, err := newRound(t.TempDir(), 0, 1)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.teardown(true))

		// Verify
		_, err = os.Stat(round.datadir)
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

// Test creating new round.
func TestRound_New(t *testing.T) {
	req := require.New(t)

	// Act
	round, err := newRound(t.TempDir(), 7, 1)
	req.NoError(err)
	t.Cleanup(func() { assert.NoError(t, round.teardown(true)) })

	// Verify
	req.EqualValues(7, round.Epoch())
	req.True(round.isOpen())
	req.False(round.isExecuted())
	req.Zero(round.executionStarted)
	req.Zero(numChallenges(round))
}

// Test submitting many challenges.
func TestRound_Submit(t *testing.T) {
	t.Run("submit many different challenges", func(t *testing.T) {
		// Arrange
		round := newTestRound(t, withMaxMembers(32))
		challenges, err := genChallenges(32)
		require.NoError(t, err)

		// Act
		for _, ch := range challenges {
			require.NoError(t, round.submit(ch, ch))
		}

		// Verify
		require.Equal(t, len(challenges), numChallenges(round))
		for _, ch := range challenges {
			challengeInDb, err := round.challengesDb.Get(ch, nil)
			require.NoError(t, err)
			require.Equal(t, ch, challengeInDb)
		}
	})
	t.Run("submit challenges with same key", func(t *testing.T) {
		// Arrange
		round := newTestRound(t)
		challenges, err := genChallenges(2)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.submit([]byte("key"), challenges[0]))
		err = round.submit([]byte("key"), challenges[1])

		// Verify
		require.ErrorIs(t, err, ErrChallengeAlreadySubmitted)
		require.Equal(t, 1, numChallenges(round))
		challenge, err := round.challengesDb.Get([]byte("key"), nil)
		require.NoError(t, err)
		require.Equal(t, challenges[0], challenge)
	})
	t.Run("cannot submit to round in execution", func(t *testing.T) {
		// Arrange
		round := newTestRound(t)
		challenge, err := genChallenge()
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		err = round.execute(ctx, time.Now().Add(time.Hour), 1, 0)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		// Act
		err = round.submit([]byte("key"), challenge)

		// Verify
		require.ErrorIs(t, err, ErrRoundIsNotOpen)
	})
	t.Run("cannot oversubmit", func(t *testing.T) {
		// Arrange
		round := newTestRound(t, withMaxMembers(2))
		challenges, err := genChallenges(3)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.submit(challenges[0], challenges[0]))
		require.NoError(t, round.submit(challenges[1], challenges[1]))
		require.ErrorIs(t, round.submit(challenges[2], challenges[2]), ErrMaxMembersReached)

		// Verify
		require.Equal(t, 2, numChallenges(round))
		for _, ch := range challenges[:2] {
			challengeInDb, err := round.challengesDb.Get(ch, nil)
			require.NoError(t, err)
			require.Equal(t, ch, challengeInDb)
		}
	})
}

// Test round execution.
func TestRound_Execute(t *testing.T) {
	req := require.New(t)

	// Arrange
	round := newTestRound(t)
	challenge, err := genChallenge()
	req.NoError(err)
	req.NoError(round.submit([]byte("key"), challenge))

	// Act
	req.NoError(round.execute(context.Background(), time.Now().Add(400*time.Millisecond), 1, 0))

	// Verify
	req.Equal(shared.T, round.execution.SecurityParam)
	req.Len(round.execution.Members, 1)
	req.NotZero(round.execution.NumLeaves)
	validateProof(t, round.execution)
}

func TestRound_StateRecovery(t *testing.T) {
	t.Run("Recover open round", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Arrange
		round, err := newRound(tmpdir, 0, 1)
		require.NoError(t, err)
		require.NoError(t, round.teardown(false))

		// Act
		recovered, err := newRound(tmpdir, 0, 1)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, recovered.teardown(false)) })
		require.NoError(t, recovered.loadState())

		// Verify
		require.True(t, recovered.isOpen())
		require.False(t, recovered.isExecuted())
	})
	t.Run("Recover open round with members", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Arrange
		challenge, err := genChallenge()
		require.NoError(t, err)
		{
			round, err := newRound(tmpdir, 0, 1)
			require.NoError(t, err)
			require.NoError(t, round.submit([]byte("key"), challenge))
			require.NoError(t, round.teardown(false))
		}

		// Act
		recovered, err := newRound(tmpdir, 0, 1)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, recovered.teardown(false)) })
		require.NoError(t, recovered.loadState())

		// Verify
		require.ErrorIs(t, recovered.submit([]byte("key-2"), challenge), ErrMaxMembersReached)
		require.Equal(t, 1, numChallenges(recovered))
		require.EqualValues(t, 1, recovered.members)
		require.True(t, recovered.isOpen())
		require.False(t, recovered.isExecuted())
	})
	t.Run("Load state of a freshly opened round", func(t *testing.T) {
		// Arrange
		round, err := newRound(t.TempDir(), 0, 1)
		t.Cleanup(func() { assert.NoError(t, round.teardown(false)) })
		require.NoError(t, err)
		require.NoError(t, round.saveState())

		// Act & verify
		require.NoError(t, round.loadState())
		require.True(t, round.isOpen())
		require.False(t, round.isExecuted())
	})
	t.Run("Recover executing round", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Arrange
		round, err := newRound(tmpdir, 0, 1)
		require.NoError(t, err)
		challenge, err := genChallenge()
		require.NoError(t, err)
		require.NoError(t, round.submit([]byte("key"), challenge))
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		require.ErrorIs(t, round.execute(ctx, time.Now().Add(time.Hour), 1, 0), context.DeadlineExceeded)
		require.NoError(t, round.teardown(false))

		// Act
		recovered, err := newRound(tmpdir, 0, 1)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, recovered.teardown(false)) })
		require.NoError(t, recovered.loadState())

		// Verify
		require.False(t, recovered.isOpen())
		require.False(t, recovered.isExecuted())
		require.NotZero(t, recovered.executionStarted)
	})
}

// TestRound_Recovery test round recovery functionality.
// The scenario proceeds as follows:
//   - Execute round and request shutdown before completion.
//   - Recover execution, and request shutdown before completion.
//   - Recover execution again, and let it complete.
func TestRound_ExecutionRecovery(t *testing.T) {
	req := require.New(t)
	tmpdir := t.TempDir()

	// Arrange
	challenges, err := genChallenges(32)
	req.NoError(err)

	// Execute round, and request shutdown before completion.
	{
		round, err := newRound(tmpdir, 1, 32)
		req.NoError(err)
		req.Zero(0, numChallenges(round))

		for _, ch := range challenges {
			req.NoError(round.submit(ch, ch))
		}

		ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer stop()
		req.ErrorIs(
			round.execute(ctx, time.Now().Add(time.Hour), 1, 0),
			context.DeadlineExceeded,
		)
		req.NoError(round.teardown(false))
	}

	// Recover round execution and request shutdown before completion.
	{
		round, err := newRound(tmpdir, 1, 32)
		req.NoError(err)
		req.Equal(len(challenges), numChallenges(round))
		req.NoError(round.loadState())

		ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer stop()
		req.ErrorIs(round.recoverExecution(ctx, time.Now().Add(time.Hour), 0), context.DeadlineExceeded)
		req.NoError(round.teardown(false))
	}

	// Recover r2 execution again, and let it complete.
	{
		round, err := newRound(tmpdir, 1, 32)
		req.NoError(err)
		req.Equal(len(challenges), numChallenges(round))
		req.NoError(round.loadState())

		req.NoError(round.recoverExecution(context.Background(), time.Now().Add(400*time.Millisecond), 0))
		validateProof(t, round.execution)
		req.NoError(round.teardown(true))
	}
}
