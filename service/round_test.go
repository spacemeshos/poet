package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/prover"
)

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

// TestRound_Recovery test round recovery functionality.
// The scenario proceeds as follows:
//   - Execute r1 as a reference round.
//   - Execute r2, and request shutdown before completion.
//   - Recover r2 execution, and request shutdown before completion.
//   - Recover r2 execution again, and let it complete.
func TestRound_Recovery(t *testing.T) {
	req := require.New(t)

	ctx, stop := context.WithCancel(context.Background())

	duration := 500 * time.Millisecond
	tmpdir := t.TempDir()

	challenges, err := genChallenges(32)
	req.NoError(err)

	// Execute r1 as a reference round.
	r1, err := newRound(tmpdir, 0)
	req.NoError(err)
	req.NoError(r1.open())
	req.Equal(0, r1.numChallenges())
	req.True(r1.isEmpty())

	for _, ch := range challenges {
		req.NoError(r1.submit(ch, ch))
	}
	req.Equal(len(challenges), r1.numChallenges())
	req.False(r1.isEmpty())

	req.NoError(r1.execute(ctx, time.Now().Add(duration), prover.LowestMerkleMinMemoryLayer))
	req.NoError(r1.teardown(true))

	// Execute r2, and request shutdown before completion.
	r2, err := newRound(tmpdir, 1)
	req.NoError(err)
	req.NoError(r2.open())
	req.Equal(0, r2.numChallenges())
	req.True(r2.isEmpty())

	for _, ch := range challenges {
		req.NoError(r2.submit(ch, ch))
	}
	req.Equal(len(challenges), r2.numChallenges())
	req.False(r2.isEmpty())

	stop()
	req.ErrorIs(r2.execute(ctx, time.Now().Add(duration), prover.LowestMerkleMinMemoryLayer), prover.ErrShutdownRequested)
	req.NoError(r2.teardown(false))

	// Recover r2 execution, and request shutdown before completion.
	ctx, stop = context.WithCancel(context.Background())
	r2recovery1, err := newRound(tmpdir, 1)
	req.NoError(err)
	req.Equal(len(challenges), r2recovery1.numChallenges())
	req.False(r2recovery1.isEmpty())

	state, err := r2recovery1.state()
	req.NoError(err)

	stop()
	req.ErrorIs(r2recovery1.recoverExecution(ctx, state.Execution, time.Now().Add(duration)), prover.ErrShutdownRequested)
	req.NoError(r2recovery1.teardown(false))

	// Recover r2 execution again, and let it complete.
	ctx, stop = context.WithCancel(context.Background())
	defer stop()
	r2recovery2, err := newRound(tmpdir, 1)
	req.NoError(err)
	req.Equal(len(challenges), r2recovery2.numChallenges())
	req.False(r2recovery2.isEmpty())
	state, err = r2recovery2.state()
	req.NoError(err)

	req.NoError(r2recovery2.recoverExecution(ctx, state.Execution, time.Now().Add(duration)))
	req.NoError(r2recovery2.teardown(true))
}

func TestRound_State(t *testing.T) {
	req := require.New(t)

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	tempdir := t.TempDir()

	// Create a new round.
	r, err := newRound(tempdir, 0)
	req.NoError(err)
	req.True(!r.isOpen())
	req.True(r.opened.IsZero())
	req.True(r.executionStarted.IsZero())
	_, err = r.proof(false)
	req.EqualError(err, "round wasn't open")

	req.Nil(r.stateCache)
	state, err := r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(r.datadir, roundStateFileBaseName)))
	req.Nil(state)

	challenges, err := genChallenges(32)
	req.NoError(err)

	req.EqualError(r.submit(challenges[0], challenges[0]), "round is not open")

	// Open the round.
	req.NoError(r.open())
	req.True(r.isOpen())
	_, err = r.proof(false)
	req.EqualError(err, "round is open")
	req.Equal(0, r.numChallenges())
	req.True(r.isEmpty())

	for _, ch := range challenges {
		req.NoError(r.submit(ch, ch))
	}
	req.Len(challenges, r.numChallenges())
	req.False(r.isEmpty())

	req.Nil(r.stateCache)
	state, err = r.state()
	req.NoError(err)
	req.NotNil(state)
	req.Equal(state, r.stateCache)

	req.True(state.isOpen())
	req.False(state.isExecuted())
	req.NotNil(state.Execution)
	req.NotZero(state.Execution.SecurityParam)
	req.Nil(state.Execution.Statement)
	req.Zero(state.Execution.NumLeaves)
	req.Nil(state.Execution.ParkedNodes)
	req.Nil(state.Execution.NIP)

	// Execute the round, and request shutdown before completion.
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()
	req.ErrorIs(r.execute(ctx, time.Now().Add(time.Hour), prover.LowestMerkleMinMemoryLayer), prover.ErrShutdownRequested)
	req.False(r.isOpen())
	req.False(r.opened.IsZero())
	req.False(r.executionStarted.IsZero())
	_, err = r.proof(false)
	req.EqualError(err, "round is executing") // TODO: support an explicit "crashed" state?

	state, err = r.state()
	req.NoError(err)
	req.NotNil(state)
	req.False(state.isOpen())
	req.False(state.isExecuted())
	req.NotNil(state.Execution)
	req.NotZero(state.Execution.SecurityParam)
	req.Len(state.Execution.Statement, 32)
	req.Greater(state.Execution.NumLeaves, uint64(0))
	req.NotNil(state.Execution.ParkedNodes)
	req.Nil(state.Execution.NIP)
	req.NoError(r.teardown(false))

	// Create a new round instance of the same round.
	ctx, stop = context.WithCancel(context.Background())
	defer stop()
	r, err = newRound(tempdir, 0)
	req.NoError(err)
	req.False(r.isOpen())
	req.True(r.opened.IsZero())
	req.True(r.executionStarted.IsZero())
	req.Len(challenges, r.numChallenges())
	req.False(r.isEmpty())
	_, err = r.proof(false)
	req.EqualError(err, "round wasn't open")

	prevState := state
	state, err = r.state()
	req.NoError(err)
	req.Equal(prevState, state)

	// Recover execution.
	req.NoError(r.recoverExecution(ctx, state.Execution, time.Now().Add(200*time.Millisecond)))

	req.False(r.executionStarted.IsZero())
	proof, err := r.proof(false)
	req.NoError(err)

	req.Equal(r.execution.NIP, proof.Proof)
	req.Equal(r.execution.Statement, proof.Statement)

	// Verify round execution state.
	state, err = r.state()
	req.NoError(err)
	req.False(state.isOpen())
	req.True(state.isExecuted())
	req.Equal(r.execution, state.Execution)

	// Trigger cleanup.
	req.NoError(r.teardown(true))

	// Verify cleanup.
	state, err = r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(r.datadir, roundStateFileBaseName)))
	req.Nil(state)
}
