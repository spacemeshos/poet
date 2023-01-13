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

func newTestRound(t *testing.T) *round {
	t.Helper()
	round, err := newRound(t.TempDir(), 7)
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
	t.Parallel()
	req := require.New(t)

	t.Run("no cleanup", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round, err := newRound(t.TempDir(), 0)
		req.NoError(err)

		// Act
		req.NoError(round.teardown(false))

		// Verify
		_, err = os.Stat(round.datadir)
		req.NoError(err)
	})

	t.Run("cleanup", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round, err := newRound(t.TempDir(), 0)
		req.NoError(err)

		// Act
		req.NoError(round.teardown(true))

		// Verify
		_, err = os.Stat(round.datadir)
		req.ErrorIs(err, os.ErrNotExist)
	})
}

// Test creating new round.
func TestRound_New(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Act
	round, err := newRound(t.TempDir(), 7)
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
	t.Parallel()
	req := require.New(t)

	t.Run("submit many different challenges", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t)
		challenges, err := genChallenges(32)
		req.NoError(err)

		// Act
		for _, ch := range challenges {
			req.NoError(round.submit(ch, ch))
		}

		// Verify
		req.Equal(len(challenges), numChallenges(round))
		for _, ch := range challenges {
			challengeInDb, err := round.challengesDb.Get(ch, nil)
			req.NoError(err)
			req.Equal(ch, challengeInDb)
		}
	})
	t.Run("submit challenges with same key", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t)
		challenges, err := genChallenges(2)
		req.NoError(err)

		// Act
		req.NoError(round.submit([]byte("key"), challenges[0]))
		err = round.submit([]byte("key"), challenges[1])

		// Verify
		req.ErrorIs(err, ErrChallengeAlreadySubmitted)
		req.Equal(1, numChallenges(round))
		challenge, err := round.challengesDb.Get([]byte("key"), nil)
		req.NoError(err)
		req.Equal(challenges[0], challenge)
	})
	t.Run("cannot submit to round in execution", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round := newTestRound(t)
		challenge, err := genChallenge()
		req.NoError(err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		err = round.execute(ctx, time.Now().Add(time.Hour), 1)
		req.ErrorIs(err, context.DeadlineExceeded)

		// Act
		err = round.submit([]byte("key"), challenge)

		// Verify
		req.ErrorIs(err, ErrRoundIsNotOpen)
	})
}

// Test round execution.
func TestRound_Execute(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Arrange
	round := newTestRound(t)
	challenge, err := genChallenge()
	req.NoError(err)
	req.NoError(round.submit([]byte("key"), challenge))

	// Act
	req.NoError(round.execute(context.Background(), time.Now().Add(10*time.Millisecond), 1))

	// Verify
	req.Equal(shared.T, round.execution.SecurityParam)
	req.Len(round.execution.Members, 1)
	req.NotZero(round.execution.NumLeaves)
	validateProof(t, round.execution)
}

func TestRound_StateRecovery(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	t.Run("Recover open round", func(t *testing.T) {
		t.Parallel()
		tmpdir := t.TempDir()

		// Arrange
		round, err := newRound(tmpdir, 0)
		req.NoError(err)
		req.NoError(round.teardown(false))

		// Act
		recovered, err := newRound(tmpdir, 0)
		req.NoError(err)
		t.Cleanup(func() { assert.NoError(t, recovered.teardown(false)) })
		req.NoError(recovered.loadState())

		// Verify
		req.True(recovered.isOpen())
		req.False(recovered.isExecuted())
	})
	t.Run("Recover executing round", func(t *testing.T) {
		t.Parallel()
		tmpdir := t.TempDir()

		// Arrange
		round, err := newRound(tmpdir, 0)
		req.NoError(err)
		challenge, err := genChallenge()
		req.NoError(err)
		req.NoError(round.submit([]byte("key"), challenge))
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		req.ErrorIs(round.execute(ctx, time.Now().Add(time.Hour), 1), context.DeadlineExceeded)
		req.NoError(round.teardown(false))

		// Act
		recovered, err := newRound(tmpdir, 0)
		req.NoError(err)
		t.Cleanup(func() { assert.NoError(t, recovered.teardown(false)) })
		req.NoError(recovered.loadState())

		// Verify
		req.False(recovered.isOpen())
		req.False(recovered.isExecuted())
		req.NotZero(recovered.executionStarted)
	})
}

// TestRound_Recovery test round recovery functionality.
// The scenario proceeds as follows:
//   - Execute round and request shutdown before completion.
//   - Recover execution, and request shutdown before completion.
//   - Recover execution again, and let it complete.
func TestRound_ExecutionRecovery(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	tmpdir := t.TempDir()

	// Arrange
	challenges, err := genChallenges(32)
	req.NoError(err)

	// Execute round, and request shutdown before completion.
	{
		round, err := newRound(tmpdir, 1)
		req.NoError(err)
		req.Zero(0, numChallenges(round))

		for _, ch := range challenges {
			req.NoError(round.submit(ch, ch))
		}

		ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer stop()
		req.ErrorIs(
			round.execute(ctx, time.Now().Add(time.Hour), 1),
			context.DeadlineExceeded,
		)
		req.NoError(round.teardown(false))
	}

	// Recover round execution and request shutdown before completion.
	{
		round, err := newRound(tmpdir, 1)
		req.NoError(err)
		req.Equal(len(challenges), numChallenges(round))
		req.NoError(round.loadState())

		ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer stop()
		req.ErrorIs(round.recoverExecution(ctx, time.Now().Add(time.Hour)), context.DeadlineExceeded)
		req.NoError(round.teardown(false))
	}

	// Recover r2 execution again, and let it complete.
	{
		round, err := newRound(tmpdir, 1)
		req.NoError(err)
		req.Equal(len(challenges), numChallenges(round))
		req.NoError(round.loadState())

		req.NoError(round.recoverExecution(context.Background(), time.Now().Add(10*time.Millisecond)))
		validateProof(t, round.execution)
		req.NoError(round.teardown(true))
	}
}
