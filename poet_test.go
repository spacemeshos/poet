package main

import (
	"context"
	"crypto/rand"
	//"fmt"
	"testing"
	"time"

	//"github.com/spacemeshos/go-spacemesh-mock/api/nmpb"
	nmIntegration "github.com/spacemeshos/go-spacemesh-mock/integration"
	"github.com/spacemeshos/poet/integration"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/utils"
	"github.com/stretchr/testify/require"
)

// the repo of the imported go-spacemesh-mock package
const mockRepo = "github.com/spacemeshos/go-spacemesh-mock"

// harnessTestCase represents a test-case which utilizes an instance
// of the Harness to exercise functionality.
type harnessTestCase struct {
	name string
	test func(h *integration.Harness, assert *require.Assertions, ctx context.Context)
}

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "submit", test: testSubmit},
}

func TestHarness(t *testing.T) {
	assert := require.New(t)
	// config poets harness
	cfg, err := integration.DefaultConfig()
	assert.NoError(err)
	cfg.N = 18
	cfg.InitialRoundDuration = (2 * time.Second).String()

	// config node mock harness
	// get go-spacemesh-mock pkg path for
	mockSrcCodePath, err := utils.GetPkgPath(mockRepo)
	assert.NoError(err)
	nmCfg, nmErr := nmIntegration.DefaultConfig(mockSrcCodePath)
	assert.NoError(nmErr)

	_ = newNMHarness(assert, nmCfg)
	h := newHarness(assert, cfg)

	defer func() {
		err := h.TearDown(true)
		// TODO when tearing down process panics
		//nmErr := nmh.TearDown()

		assert.NoError(err, "failed to tear down harness")
		t.Logf("harness teared down")

		//assert.NoError(nmErr, "failed to tear down node mock")
		//t.Logf("node mock teared down")
	}()
	// (amit)same assertion as in line 34
	assert.NoError(err)
	// (amit)already asserted in newHarness func
	assert.NotNil(h)
	t.Logf("harness launched")
	//ctx := context.Background()
	//fmt.Println("submit")
	//_, err = h.Submit(ctx, &api.SubmitRequest{Challenge: []byte("this is a commitment")})
	//assert.NoError(err, "rpc error: code = Unknown desc = service not started")
	////assert.EqualError(err, "rpc error: code = Unknown desc = service not started")
	//
	//fmt.Println("start")
	//_, err = h.Start(ctx, &api.StartRequest{NodeAddress: cfg.NodeAddress})
	//assert.EqualError(err, "rpc error: code = Unknown desc = failed to start service: failed not connect to gateway node (addr: 666): failed to connect to rpc server: context deadline exceeded")
	//
	//fmt.Println("start2")
	//_, err = h.Start(ctx, &api.StartRequest{NodeAddress: cfg.NodeAddress}) // cfg.NodeAddress
	//assert.EqualError(err, "rpc error: code = Unknown desc = failed to start service: already opened")
	////assert.NoError(err)
	//
	//fmt.Println("start3")
	//_, err = h.Start(ctx, &api.StartRequest{NodeAddress: "NO_BROADCAST"})
	//assert.EqualError(err, "rpc error: code = Unknown desc = failed to start service: already opened")
	//fmt.Println("final")

	for _, testCase := range testCases {
		success := t.Run(testCase.name, func(t1 *testing.T) {
			ctx, _ := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
			testCase.test(h, assert, ctx)
		})

		if !success {
			break
		}
	}

}

func testInfo(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	// TODO: implement
	_, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
}

func testSubmit(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	com := []byte("this is a commitment")
	submitReq := api.SubmitRequest{Challenge: com}
	submitRes, err := h.Submit(ctx, &submitReq)
	assert.NoError(err)
	assert.NotNil(submitRes)
}

// TestHarness_CrashRecovery test the server recovery functionality.
// The scenario proceeds as follows:
//  - Generate a set of challenges.
// 	- Create a server harness instance, and submit challenges to his rounds (0 and 1).
//  - TODO: Wait until rounds 0 and 1 execution completes. listen to their proof broadcast and save it as a reference.
//  - Tear down the server with disk cleanup.
// 	- Create a new server harness instance, and submit challenges to his rounds (0 and 1).
//  - Let round 0 execution to proceed a bit.
//  - Tear down the server without disk cleanup.
//  - Create a new server harness instance
//  - TODO: Wait until rounds 0 and 1 recovery completes. listen to their proof broadcast and compare it with the reference rounds.
func tTestHarness_CrashRecovery(t *testing.T) {
	req := require.New(t)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(30*time.Second))

	cfg, err := integration.DefaultConfig()
	req.NoError(err)
	cfg.N = 18
	cfg.InitialRoundDuration = time.Duration(1 * time.Second).String()

	// Track rounds.
	numRounds := 2
	roundsId := make([]string, numRounds)

	// Generate random challenges for 2 distinct rounds.
	numRoundChallenges := 40
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

			// Verify that all submissions returned the same round instance.
			if roundsId[roundIndex] == "" {
				roundsId[roundIndex] = res.RoundId
			} else {
				req.Equal(roundsId[roundIndex], res.RoundId)
			}
		}
	}

	waitNewRound := func(h *integration.Harness, currentRoundId string) {
		timeout := time.After(5 * time.Second)
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
	h := newHarness(req, cfg)

	// Submit challenges to open round (0).
	submitChallenges(h, 0)

	// Verify that round 0 is still open.
	info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(roundsId[0], info.OpenRoundId)

	// Wait until round iteration proceeds: a new round opened, previous round is executing.
	waitNewRound(h, roundsId[0])

	// Submit challenges to open round (1).
	submitChallenges(h, 1)

	// Verify that round 1 is still open, and round 0 is executing.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(1, len(info.ExecutingRoundsIds))
	req.Equal(roundsId[0], info.ExecutingRoundsIds[0])
	req.Equal(roundsId[1], info.OpenRoundId)

	// TODO: Wait until rounds 0 and 1 execution completes. listen to their proof broadcast and save it as a reference.
	time.Sleep(10 * time.Second) // remove, used for manual log check

	// Tear down the server with disk cleanup.
	err = h.TearDown(true)
	req.NoError(err)

	// Create a new server harness instance.
	h = newHarness(req, cfg)
	roundsId = make([]string, numRounds)

	// Submit challenges to open round (0).
	submitChallenges(h, 0)

	// Verify that round 0 is still open.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(roundsId[0], info.OpenRoundId)

	// Wait until round iteration proceeds: a new round opened, previous round is executing.
	waitNewRound(h, roundsId[0])

	// Submit challenges to open round (1).
	submitChallenges(h, 1)

	// Verify that round 1 is still open, and round 0 is executing.
	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	req.NoError(err)
	req.Equal(1, len(info.ExecutingRoundsIds))
	req.Equal(roundsId[0], info.ExecutingRoundsIds[0])
	req.Equal(roundsId[1], info.OpenRoundId)

	// Wait a bit for round 0 execution to proceed.
	time.Sleep(1 * time.Second)

	// Tear down the server without disk cleanup (ungraceful shutdown).
	err = h.TearDown(false)
	req.NoError(err)

	// Launch another server, with the same config.
	h = newHarness(req, cfg)

	// TODO: wait until both rounds 0 and 1 recovery completes. listen to their proof broadcast and compare it with the reference rounds.
	// with the reference rounds.
	time.Sleep(10 * time.Second) // remove. used for manual log check

	err = h.TearDown(true)
	req.NoError(err)
}

func newHarness(req *require.Assertions, cfg *integration.ServerConfig) *integration.Harness {
	h, err := integration.NewHarness(cfg)
	req.NoError(err)
	req.NotNil(h)

	go func() {
		for {
			err, more := <-h.ProcessErrors()
			if !more {
				return
			}
			req.Fail("poet server finished with error", err)
		}
	}()

	return h
}

func newNMHarness(req *require.Assertions, cfg *nmIntegration.ServerConfig) *nmIntegration.Harness {
	h, err := nmIntegration.NewHarness(cfg)
	req.NoError(err)
	req.NotNil(h)

	go func() {
		for {
			err, more := <-h.ProcessErrors()
			if !more {
				return
			}
			req.Fail("go spacemesh mock server has finished with error", err)
		}
	}()

	return h
}

type challenge struct {
	data    []byte
	roundId int
}
