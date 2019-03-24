package integration

import (
	"context"
	"github.com/spacemeshos/poet-ref/rpc/api"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// harnessTestCase represents a test-case which utilizes an instance
// of the Harness to exercise functionality.
type harnessTestCase struct {
	name string
	test func(h *Harness, assert *require.Assertions, ctx context.Context)
}

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
}

func TestHarness(t *testing.T) {
	assert := require.New(t)
	h, err := NewHarness()

	go func() {
		for {
			select {
			case err, more := <-h.ProcessErrors():
				if !more {
					return
				}
				t.Logf("poet server finished with error (stderr):\n%v", err)
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

func testInfo(h *Harness, assert *require.Assertions, ctx context.Context) {
	// TODO: implement
	_, err := h.client.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
}
