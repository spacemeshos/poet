package service

import (
	"fmt"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/signal"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

var (
	recoveryExecDecreaseThreshold = 0.85
)

// TestRound_Recovery test round recovery functionality.
// The scenario proceeds as follows:
// 	- Execute r1 as a reference round.
//  - Execute r2, and request shutdown before completion.
//  - Recover r2 execution, and request shutdown before completion.
//  - Recover r2 execution again, and let it complete.
//  - Compare r2 total execution time and execution results with r1.
func TestRound_Recovery(t *testing.T) {
	req := require.New(t)

	sig := signal.NewSignal()
	cfg := &Config{N: 18}

	challenges, err := genChallenges(32)
	req.NoError(err)

	// Execute r1 as a reference round.
	tempdir, _ := ioutil.TempDir("", "poet-test")
	r1 := newRound(sig, cfg, tempdir, "test-round-1")
	req.NoError(r1.open())
	req.Equal(0, r1.numChallenges())
	req.True(r1.isEmpty())

	for _, ch := range challenges {
		req.NoError(r1.submit(ch))
	}
	req.Equal(len(challenges), r1.numChallenges())
	req.False(r1.isEmpty())

	start := time.Now()
	req.NoError(r1.execute())
	r1exec := time.Since(start)

	// Execute r2, and request shutdown before completion.
	tempdir, _ = ioutil.TempDir("", "poet-test")
	r2 := newRound(sig, cfg, tempdir, "test-round-2")
	req.NoError(r2.open())
	req.Equal(0, r2.numChallenges())
	req.True(r2.isEmpty())

	for _, ch := range challenges {
		req.NoError(r2.submit(ch))
	}
	req.Equal(len(challenges), r2.numChallenges())
	req.False(r2.isEmpty())

	go func() {
		time.Sleep(r1exec / 3)
		sig.RequestShutdown()
	}()

	start = time.Now()
	req.EqualError(r2.execute(), prover.ErrShutdownRequested.Error())
	r2exec1 := time.Since(start)

	// Recover r2 execution, and request shutdown before completion.
	sig = signal.NewSignal()
	r2recovery1 := newRound(sig, cfg, tempdir, "test-round-2-recovery-1")
	req.Equal(len(challenges), r2recovery1.numChallenges())
	req.False(r2recovery1.isEmpty())

	state, err := r2recovery1.state()
	req.NoError(err)

	go func() {
		time.Sleep(r1exec / 3)
		sig.RequestShutdown()
	}()

	start = time.Now()
	req.EqualError(r2recovery1.recoverExecution(state.Execution), prover.ErrShutdownRequested.Error())
	r2exec2 := time.Since(start)

	// Recover r2 execution again, and let it complete.
	sig = signal.NewSignal()
	r2recovery2 := newRound(sig, cfg, tempdir, "test-round-2-recovery-2")
	req.Equal(len(challenges), r2recovery2.numChallenges())
	req.False(r2recovery2.isEmpty())
	state, err = r2recovery2.state()
	req.NoError(err)

	start = time.Now()
	req.NoError(r2recovery2.recoverExecution(state.Execution))
	r2exec3 := time.Since(start)

	// Compare r2 total execution time and execution results with r1.
	r2exec := r2exec1 + r2exec2 + r2exec3
	diff := float64(r1exec) / float64(r2exec)
	//req.True(diff > recoveryExecDecreaseThreshold, fmt.Sprintf("recovery execution time comparison is below the threshold: %f", diff))
	t.Logf("recovery execution time diff: %f", diff)

	req.Equal(r1.execution.NIP, r2recovery2.execution.NIP)
}

func TestRound_State(t *testing.T) {
	req := require.New(t)

	sig := signal.NewSignal()
	cfg := &Config{N: 18}
	tempdir, _ := ioutil.TempDir("", "poet-test")

	// Create a new round.
	r := newRound(sig, cfg, tempdir, "test-round")
	req.True(!r.isOpen())
	req.True(r.opened.IsZero())
	req.True(r.executionStarted.IsZero())
	_, err := r.proof(false)
	req.EqualError(err, "round wasn't open")

	req.Nil(r.stateCache)
	state, err := r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(tempdir, roundStateFileBaseName)))
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
	req.True(!state.Opened.IsZero())
	req.True(state.ExecutionStarted.IsZero())
	req.NotNil(state.Execution)
	req.True(state.Execution.NumLeaves != 0)
	req.True(state.Execution.SecurityParam != 0)
	req.True(state.Execution.Statement == nil)
	req.True(state.Execution.NextLeafId == 0)
	req.True(state.Execution.ParkedNodes == nil)
	req.True(state.Execution.NIP == nil)

	// Execute the round, and request shutdown before completion.
	go func() {
		time.Sleep(100 * time.Millisecond)
		sig.RequestShutdown()
	}()

	req.EqualError(r.execute(), prover.ErrShutdownRequested.Error())
	req.True(!r.isOpen())
	req.True(!r.opened.IsZero())
	req.True(!r.executionStarted.IsZero())
	_, err = r.proof(false)
	req.EqualError(err, "round is executing") // TODO: support an explicit "crashed" state?

	state, err = r.state()
	req.NoError(err)
	req.NotNil(state)
	req.True(!state.isOpen())
	req.True(!state.Opened.IsZero())
	req.True(!state.ExecutionStarted.IsZero())
	req.NotNil(state.Execution)
	req.True(state.Execution.NumLeaves != 0)
	req.True(state.Execution.SecurityParam != 0)
	req.True(len(state.Execution.Statement) == 32)
	req.True(state.Execution.NextLeafId > 0)
	req.True(state.Execution.ParkedNodes != nil)
	req.True(state.Execution.NIP == nil)

	// Create a new round instance of the same round.
	r = newRound(signal.NewSignal(), cfg, tempdir, "test-round")
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
	req.NoError(r.recoverExecution(state.Execution))

	req.True(!r.executionStarted.IsZero())
	proof, err := r.proof(false)
	req.NoError(err)

	req.Equal(r.execution.NIP, proof.Proof)
	req.Equal(r.execution.Statement, proof.Statement)

	// Verify round cleanup.
	time.Sleep(1 * time.Second)
	state, err = r.state()
	req.EqualError(err, fmt.Sprintf("file is missing: %v", filepath.Join(tempdir, roundStateFileBaseName)))
	req.Nil(state)
}
