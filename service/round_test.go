package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/state"
	"github.com/spacemeshos/poet/verifier"
)

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
	t.Run("no cleanup", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round, err := NewRound(t.TempDir(), 0)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.Teardown(context.Background(), false))

		// Verify
		_, err = os.Stat(round.datadir)
		require.NoError(t, err)
	})

	t.Run("cleanup", func(t *testing.T) {
		t.Parallel()
		// Arrange
		round, err := NewRound(t.TempDir(), 0)
		require.NoError(t, err)

		// Act
		require.NoError(t, round.Teardown(context.Background(), true))

		// Verify
		_, err = os.Stat(round.datadir)
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

// Test creating new round.
func TestRound_New(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Act
	round, err := NewRound(t.TempDir(), 7)
	req.NoError(err)
	t.Cleanup(func() { assert.NoError(t, round.Teardown(context.Background(), true)) })

	// Verify
	req.EqualValues(7, round.epoch)
	req.False(round.IsFinished())
	req.Zero(round.executionStarted)
}

// Test round execution.
func TestRound_Execute(t *testing.T) {
	req := require.New(t)

	// Arrange
	round, err := NewRound(t.TempDir(), 77, WithMembershipRoot([]byte("root")))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, round.Teardown(context.Background(), true)) })

	// Act
	req.NoError(round.Execute(context.Background(), time.Now().Add(400*time.Millisecond), 1, 0))

	// Verify
	req.Equal(uint(77), round.epoch)
	req.Equal([]byte("root"), round.execution.Statement)
	req.Equal(shared.T, round.execution.SecurityParam)
	req.NotZero(round.execution.NumLeaves)
	validateProof(t, round.execution)
}

func TestRound_StateRecovery(t *testing.T) {
	t.Parallel()
	t.Run("Recover open round", func(t *testing.T) {
		t.Parallel()
		tmpdir := t.TempDir()

		// Arrange
		round, err := NewRound(tmpdir, 0)
		require.NoError(t, err)
		require.NoError(t, round.Teardown(context.Background(), false))

		// Act
		recovered, err := NewRound(tmpdir, 0)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, recovered.Teardown(context.Background(), false)) })

		// Verify
		require.False(t, recovered.IsFinished())
	})
	t.Run("Recover executing round", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Arrange
		round, err := NewRound(tmpdir, 0)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		require.ErrorIs(t, round.Execute(ctx, time.Now().Add(time.Hour), 1, 0), context.DeadlineExceeded)
		require.NoError(t, round.Teardown(context.Background(), false))

		// Act
		recovered, err := NewRound(tmpdir, 0)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, recovered.Teardown(context.Background(), false)) })

		// Verify
		require.False(t, recovered.IsFinished())
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

	// Execute round, and request shutdown before completion.
	{
		round, err := NewRound(tmpdir, 1)
		req.NoError(err)

		ctx, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer stop()
		req.ErrorIs(
			round.Execute(ctx, time.Now().Add(time.Hour), 2, 0),
			context.DeadlineExceeded,
		)
		req.NoError(round.Teardown(context.Background(), false))
	}

	// Recover round execution and request shutdown before completion.
	{
		round, err := NewRound(tmpdir, 1)
		req.NoError(err)

		ctx, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer stop()
		req.ErrorIs(round.RecoverExecution(ctx, time.Now().Add(time.Hour), 0), context.DeadlineExceeded)
		req.NoError(round.Teardown(context.Background(), false))
	}

	// Recover r2 execution again, and let it complete.
	{
		round, err := NewRound(tmpdir, 1)
		req.NoError(err)

		req.NoError(round.RecoverExecution(context.Background(), time.Now().Add(400*time.Millisecond), 0))
		validateProof(t, round.execution)
		req.NoError(round.Teardown(context.Background(), true))
	}
}

func Test_FixRoundState(t *testing.T) {
	filename := "fill path to the round's state.bin. Usually '/home/<username>/.cache/poet/data/rounds/11/state.bin'. Create a backup first."
	updatedFilename := filename + "-updated"

	s := &roundState{}
	require.NoError(t, state.Load(filename, s))

	// Fill in desired value to shorten the tree
	s.Execution.NumLeaves = 47971072
	require.NoError(t, state.Persist(updatedFilename, s))
	fmt.Printf("New state saved to %s\n", updatedFilename)
}
