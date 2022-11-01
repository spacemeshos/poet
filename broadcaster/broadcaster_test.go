package broadcaster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	req := require.New(t)

	connCtx, cancel := context.WithTimeout(context.Background(), DefaultConnTimeout)
	defer cancel()

	b, err := New(connCtx, []string(nil), true, 0, DefaultBroadcastTimeout, 0)
	req.NotNil(b)
	req.NoError(err)

	b, err = New(connCtx, []string(nil), false, 0, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses must be greater than 0")

	b, err = New(connCtx, make([]string, 0), false, 0, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses must be greater than 0")

	b, err = New(connCtx, make([]string, 1), false, 0, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful connections threshold must be greater than 0")

	b, err = New(connCtx, make([]string, 1), false, 1, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful broadcast threshold must be greater than 0")

	b, err = New(connCtx, make([]string, 1), false, 2, DefaultBroadcastTimeout, 1)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses (1) must be greater than the successful connections threshold (2)")

	b, err = New(connCtx, make([]string, 1), false, 1, DefaultBroadcastTimeout, 2)
	req.Nil(b)
	req.EqualError(err, "the successful connections threshold (1) must be greater than the successful broadcast threshold (2)")

	connCtx, cancel = context.WithTimeout(context.Background(), 0)
	defer cancel()

	b, err = New(connCtx, []string{"666"}, false, 1, 0, 1)
	req.Nil(b)
	req.EqualError(err, "failed to connect to gateway grpc server 666 (context deadline exceeded)")

	b, err = New(connCtx, []string{"666", "667"}, false, 1, 0, 1)
	req.Nil(b)
	req.EqualError(err, "failed to connect to gateway grpc server 666 (context deadline exceeded) | failed to connect to gateway grpc server 667 (context deadline exceeded)")
}
