package main

import (
	"context"
	"github.com/spacemeshos/poet-ref/integration"
	"github.com/spacemeshos/poet-ref/rpc/api"
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

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "membership proof", test: testMembershipProof},
	{name: "proof", test: testProof},
}

func TestHarness(t *testing.T) {
	assert := require.New(t)

	h, err := integration.NewHarness()
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

func testMembershipProof(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	com := []byte("this is a commitment")
	submitReq := api.SubmitRequest{Challenge: com}
	submitRes, err := h.Submit(ctx, &submitReq)
	assert.NoError(err)
	assert.NotNil(submitRes)

	mProofReq := api.GetMembershipProofRequest{RoundId: submitRes.RoundId, Commitment: com, Wait: false}
	mProofRes, err := h.GetMembershipProof(ctx, &mProofReq)
	assert.EqualError(err, "rpc error: code = Unknown desc = round is open")
	assert.Nil(mProofRes)

	mProofReq = api.GetMembershipProofRequest{RoundId: submitRes.RoundId, Commitment: com, Wait: true}
	mProofRes, err = h.GetMembershipProof(ctx, &mProofReq)
	assert.NoError(err)
	assert.NotNil(mProofRes)

	// TODO(moshababo): assert the proof verification
}

func testProof(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	com := []byte("this is a commitment")
	submitReq := api.SubmitRequest{Challenge: com}
	submitRes, err := h.Submit(ctx, &submitReq)
	assert.NoError(err)
	assert.NotNil(submitRes)

	proofReq := api.GetProofRequest{RoundId: submitRes.RoundId, Wait: false}
	proofRes, err := h.GetProof(ctx, &proofReq)
	assert.EqualError(err, "rpc error: code = Unknown desc = round is open")
	assert.Nil(proofRes)

	proofReq = api.GetProofRequest{RoundId: submitRes.RoundId, Wait: true}
	proofRes, err = h.GetProof(ctx, &proofReq)
	assert.NoError(err)
	assert.NotNil(proofRes)
	assert.NotNil(proofRes.Proof)

	// TODO(moshababo): assert the proof verification
}
