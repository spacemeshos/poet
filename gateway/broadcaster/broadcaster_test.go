package broadcaster

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNew(t *testing.T) {
	req := require.New(t)

	b, err := New([]*grpc.ClientConn(nil), true, 0, DefaultBroadcastTimeout, 0)
	req.NotNil(b)
	req.NoError(err)

	b, err = New([]*grpc.ClientConn(nil), false, 0, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "number of gateway connections must be greater than 0")

	b, err = New(make([]*grpc.ClientConn, 1), false, 0, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful connections threshold must be greater than 0")

	b, err = New(make([]*grpc.ClientConn, 1), false, 1, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful broadcast threshold must be greater than 0")

	b, err = New(make([]*grpc.ClientConn, 1), false, 2, DefaultBroadcastTimeout, 1)
	req.Nil(b)
	req.EqualError(err, "number of gateway connections (1) must be greater than the successful connections threshold (2)")

	b, err = New(make([]*grpc.ClientConn, 1), false, 1, DefaultBroadcastTimeout, 2)
	req.Nil(b)
	req.EqualError(err, "the successful connections threshold (1) must be greater than the successful broadcast threshold (2)")
}
