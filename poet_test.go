package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/integration"
	"github.com/spacemeshos/poet/rpc/api"
)

// harnessTestCase represents a test-case which utilizes an instance
// of the Harness to exercise functionality.
type harnessTestCase struct {
	name string
	test func(ctx context.Context, h *integration.Harness, assert *require.Assertions)
}

// TODO(moshababo): create a mock for the node which the harness poet server
// is broadcasting to. Without it, since the latest API change,
// these tests are quite meaningless.

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "submit", test: testSubmit},
}

func TestHarness(t *testing.T) {
	r := require.New(t)

	cfg, err := integration.DefaultConfig()
	r.NoError(err)
	cfg.Genesis = time.Now()

	h := newHarness(t, cfg)
	t.Cleanup(func() {
		err := h.TearDown(true)
		if assert.NoError(t, err, "failed to tear down harness") {
			t.Logf("harness torn down")
		}
	})

	r.NoError(err)
	r.NotNil(h)
	t.Logf("harness launched")

	ctx := context.Background()
	_, err = h.Submit(ctx, &api.SubmitRequest{Challenge: []byte("this is a commitment")})
	r.EqualError(err, "rpc error: code = Unknown desc = service not started")

	_, err = h.Start(ctx, &api.StartRequest{GatewayAddresses: []string{"666"}})
	r.EqualError(err, "rpc error: code = Unknown desc = failed to connect to Spacemesh gateway node at \"666\": failed to connect to rpc server: context deadline exceeded")

	_, err = h.Start(ctx, &api.StartRequest{DisableBroadcast: true})
	r.NoError(err)

	_, err = h.Start(ctx, &api.StartRequest{DisableBroadcast: true})
	r.EqualError(err, "rpc error: code = Unknown desc = already started")

	for _, testCase := range testCases {
		success := t.Run(testCase.name, func(t1 *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
			defer cancel()

			testCase.test(ctx, h, r)
		})

		if !success {
			break
		}
	}
}

func testInfo(ctx context.Context, h *integration.Harness, assert *require.Assertions) {
	// TODO: implement
	_, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
}

func testSubmit(ctx context.Context, h *integration.Harness, assert *require.Assertions) {
	com := []byte("this is a commitment")
	submitReq := api.SubmitRequest{Challenge: com}
	submitRes, err := h.Submit(ctx, &submitReq)
	assert.NoError(err)
	assert.NotNil(submitRes)
}

func TestHarness_CrashRecovery(t *testing.T) {
	req := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*time.Second))
	defer cancel()

	cfg, err := integration.DefaultConfig()
	req.NoError(err)
	cfg.Genesis = time.Now().Add(1 * time.Second)
	cfg.Reset = true
	cfg.DisableBroadcast = true

	// Track rounds.
	numRounds := 40
	roundsID := make([]string, 0, numRounds)
	for i := 0; i < numRounds; i++ {
		roundsID = append(roundsID, strconv.Itoa(i))
	}

	numRoundChallenges := 2
	roundsChallenges := make([][]challenge, numRounds)
	for i := 0; i < numRounds; i++ {
		challenges := make([]challenge, numRoundChallenges)
		for i := 0; i < numRoundChallenges; i++ {
			challenges[i] = challenge{data: make([]byte, 32)}
			_, err := rand.Read(challenges[i].data)
			req.NoError(err)
		}
		roundsChallenges[i] = challenges
	}

	submitChallenges := func(h *integration.Harness, roundIndex int) {
		roundChallenges := roundsChallenges[roundIndex]
		for i := 0; i < len(roundChallenges); i++ {
			res, err := h.Submit(ctx, &api.SubmitRequest{Challenge: roundChallenges[i].data})
			req.NoError(err)
			req.NotNil(res)
			req.Equalf(roundsID[roundIndex], res.RoundId, "round id mismatch for challenge %d", i)
		}
	}

	waitNewRound := func(h *integration.Harness, currentRoundId string) {
		timeout := time.After(cfg.EpochDuration)
		isOpenRound := func(val string) bool {
			info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
			req.NoError(err)
			return info.OpenRoundId == val
		}
		for isOpenRound(currentRoundId) {
			select {
			case <-timeout:
				req.Fail("round iteration timeout")
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	// Create a server harness instance.
	h := newHarness(t, cfg)

	// Submit challenges to open round (0).
	submitChallenges(h, 0)

	// Verify that round 0 is still open.
	info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(roundsID[0], info.OpenRoundId)

	// Wait until round iteration proceeds: a new round opened, previous round is executing.
	waitNewRound(h, roundsID[0])

	// Submit challenges to open round (1).
	submitChallenges(h, 1)

	// Verify that round 1 is still open, and round 0 is executing.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(1, len(info.ExecutingRoundsIds))
	req.Equal(roundsID[0], info.ExecutingRoundsIds[0])
	req.Equal(roundsID[1], info.OpenRoundId)

	// TODO: Wait until rounds 0 and 1 execution completes. listen to their proof broadcast and save it as a reference.
	req.NoError(h.TearDown(false))

	// Create a new server harness instance.
	h = newHarness(t, cfg)

	// Verify that round 1 is still open.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)

	req.Equal(roundsID[1], info.OpenRoundId)

	// Wait until round iteration proceeds: a new round opened, previous round is executing.
	waitNewRound(h, roundsID[1])

	// Submit challenges to open round (1).
	submitChallenges(h, 2)

	// Verify that round 1 is still open, and round 0 is executing.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(1, len(info.ExecutingRoundsIds))
	req.Equal(roundsID[1], info.ExecutingRoundsIds[0])
	req.Equal(roundsID[2], info.OpenRoundId)
	req.NoError(h.TearDown(true))

	// leave some time for the server to shutdown.
	time.Sleep(100 * time.Millisecond)
}

func newHarness(tb testing.TB, cfg *integration.ServerConfig) *integration.Harness {
	h, err := integration.NewHarness(cfg)
	require.NoError(tb, err)
	require.NotNil(tb, h)

	go func() {
		for {
			err, more := <-h.ProcessErrors()
			if !more {
				return
			}
			require.Fail(tb, "poet server finished with error", err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(h.StderrPipe())
		for scanner.Scan() {
			tb.Logf("stderr: %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(h.StdoutPipe())
		for scanner.Scan() {
			tb.Logf("stdout: %s", scanner.Text())
		}
	}()

	return h
}

type challenge struct {
	data []byte
}
