package main

import (
	"context"
	"github.com/spacemeshos/poet/integration"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// harnessTestCase represents a test-case which utilizes an instance
// of the Harness to exercise functionality.
type harnessTestCase struct {
	name string
	test func(h *integration.Harness, assert *require.Assertions, ctx context.Context)
}

// TODO(moshababo): create a mock for the node which the harness poet server
// is broadcasting to. Without it, since the latest API change,
// these tests are quite meaningless.

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "submit", test: testSubmit},
}

func TestHarness(t *testing.T) {
	assert := require.New(t)

	cfg, err := integration.DefaultConfig()
	assert.NoError(err)
	cfg.NodeAddress = "NO_BROADCAST"
	cfg.InitialRoundDuration = time.Duration(2 * time.Second).String()
	h, err := integration.NewHarness(cfg)
	assert.NoError(err)

	go func() {
		for {
			select {
			case err, more := <-h.ProcessErrors():
				if !more {
					return
				}
				assert.Fail("poet server finished with error", err)
			}
		}
	}()

	defer func() {
		err := h.TearDown()
		assert.NoError(err, "failed to tear down harness")
		t.Logf("harness teared down")
	}()

	assert.NoError(err)
	assert.NotNil(h)
	t.Logf("harness launched")

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
