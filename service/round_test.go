package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/signal"
)

// TestRound_Recovery test round recovery functionality.
// The scenario proceeds as follows:
//   - Execute r1 as a reference round.
//   - Execute r2, and request shutdown before completion.
//   - Recover r2 execution, and request shutdown before completion.
//   - Recover r2 execution again, and let it complete.
func TestRound_Recovery(t *testing.T) {
	req := require.New(t)

	sig := signal.NewSignal()
	duration := 500 * time.Millisecond
	tmpdir := t.TempDir()

	challenges, err := genChallenges(32)
	req.NoError(err)

	// Execute r1 as a reference round.
	r1 := newRound(sig, tmpdir, 0)
	req.NoError(r1.open())
	req.Equal(0, r1.numChallenges())
	req.True(r1.isEmpty())

	for _, ch := range challenges {
		req.NoError(r1.submit(ch))
	}
	req.Equal(len(challenges), r1.numChallenges())
	req.False(r1.isEmpty())

	req.NoError(r1.execute(time.Now().Add(duration), prover.LowestMerkleMinMemoryLayer))

	// Execute r2, and request shutdown before completion.
	r2 := newRound(sig, tmpdir, 1)
	req.NoError(r2.open())
	req.Equal(0, r2.numChallenges())
	req.True(r2.isEmpty())

	for _, ch := range challenges {
		req.NoError(r2.submit(ch))
	}
	req.Equal(len(challenges), r2.numChallenges())
	req.False(r2.isEmpty())

	go func() {
		time.Sleep(duration / 10)
		sig.RequestShutdown()
	}()

	req.ErrorIs(r2.execute(time.Now().Add(duration), prover.LowestMerkleMinMemoryLayer), prover.ErrShutdownRequested)
	require.NoError(t, r2.waitTeardown(context.TODO()))

	// Recover r2 execution, and request shutdown before completion.
	sig = signal.NewSignal()
	r2recovery1 := newRound(sig, tmpdir, 1)
	req.Equal(len(challenges), r2recovery1.numChallenges())
	req.False(r2recovery1.isEmpty())

	state, err := r2recovery1.state()
	req.NoError(err)

	go func() {
		time.Sleep(duration / 5)
		sig.RequestShutdown()
	}()

	req.ErrorIs(r2recovery1.recoverExecution(state.Execution, time.Now().Add(duration)), prover.ErrShutdownRequested)
	require.NoError(t, r2recovery1.waitTeardown(context.TODO()))

	// Recover r2 execution again, and let it complete.
	sig = signal.NewSignal()
	r2recovery2 := newRound(sig, tmpdir, 1)
	req.Equal(len(challenges), r2recovery2.numChallenges())
	req.False(r2recovery2.isEmpty())
	state, err = r2recovery2.state()
	req.NoError(err)

	req.NoError(r2recovery2.recoverExecution(state.Execution, time.Now().Add(duration)))

	// Request shutdown.
	sig.RequestShutdown()
	time.Sleep(100 * time.Millisecond)
}

func TestRound_State(t *testing.T) {
	req := require.New(t)

	sig := signal.NewSignal()
	tempdir := t.TempDir()

	// Create a new round.
	r := newRound(sig, tempdir, 0)
	req.True(!r.isOpen())
	req.True(r.opened.IsZero())
	req.True(r.executionStarted.IsZero())
	_, err := r.proof(false)
	req.EqualError(err, "round wasn't open")

	req.Nil(r.stateCache)
	state, err := r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(r.datadir, roundStateFileBaseName)))
	req.Nil(state)

	challenges, err := genChallenges(32)
	req.NoError(err)

	req.EqualError(r.submit(challenges[0]), "round is not open")

	// Open the round.
	req.NoError(r.open())
	req.True(r.isOpen())
	_, err = r.proof(false)
	req.EqualError(err, "round is open")
	req.Equal(0, r.numChallenges())
	req.True(r.isEmpty())

	for _, ch := range challenges {
		req.NoError(r.submit(ch))
	}
	req.Equal(len(challenges), r.numChallenges())
	req.False(r.isEmpty())

	req.Nil(r.stateCache)
	state, err = r.state()
	req.NoError(err)
	req.NotNil(state)
	req.Equal(state, r.stateCache)

	req.True(state.isOpen())
	req.True(!state.isExecuted())
	req.NotNil(state.Execution)
	req.True(state.Execution.SecurityParam != 0)
	req.True(state.Execution.Statement == nil)
	req.True(state.Execution.NumLeaves == 0)
	req.True(state.Execution.ParkedNodes == nil)
	req.True(state.Execution.NIP == nil)

	// Execute the round, and request shutdown before completion.
	duration := 100 * time.Millisecond
	go func() {
		time.Sleep(duration / 5)
		sig.RequestShutdown()
	}()

	req.ErrorIs(r.execute(time.Now().Add(duration), prover.LowestMerkleMinMemoryLayer), prover.ErrShutdownRequested)
	req.True(!r.isOpen())
	req.True(!r.opened.IsZero())
	req.True(!r.executionStarted.IsZero())
	_, err = r.proof(false)
	req.EqualError(err, "round is executing") // TODO: support an explicit "crashed" state?

	state, err = r.state()
	req.NoError(err)
	req.NotNil(state)
	req.True(!state.isOpen())
	req.True(!state.isExecuted())
	req.NotNil(state.Execution)
	req.True(state.Execution.SecurityParam != 0)
	req.True(len(state.Execution.Statement) == 32)
	req.True(state.Execution.NumLeaves > 0)
	req.True(state.Execution.ParkedNodes != nil)
	req.True(state.Execution.NIP == nil)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	require.NoError(t, r.waitTeardown(ctx))

	// Create a new round instance of the same round.
	r = newRound(signal.NewSignal(), tempdir, 0)
	req.True(!r.isOpen())
	req.True(r.opened.IsZero())
	req.True(r.executionStarted.IsZero())
	req.Equal(len(challenges), r.numChallenges())
	req.False(r.isEmpty())
	_, err = r.proof(false)
	req.EqualError(err, "round wasn't open")

	prevState := state
	state, err = r.state()
	req.NoError(err)
	req.Equal(prevState, state)

	// Recover execution.
	req.NoError(r.recoverExecution(state.Execution, time.Now().Add(100*time.Microsecond)))

	req.True(!r.executionStarted.IsZero())
	proof, err := r.proof(false)
	req.NoError(err)

	req.Equal(r.execution.NIP, proof.Proof)
	req.Equal(r.execution.Statement, proof.Statement)

	// Verify round execution state.
	state, err = r.state()
	req.NoError(err)
	req.True(!state.isOpen())
	req.True(state.isExecuted())
	req.Equal(r.execution, state.Execution)

	// Trigger cleanup.
	r.broadcasted()
	time.Sleep(1 * time.Second)

	// Verify cleanup.
	state, err = r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(r.datadir, roundStateFileBaseName)))
	req.Nil(state)
}
