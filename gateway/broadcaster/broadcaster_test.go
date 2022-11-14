package broadcaster

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNew(t *testing.T) {
	req := require.New(t)

	b, err := New([]*grpc.ClientConn(nil), true, DefaultBroadcastTimeout, 0)
	req.NotNil(b)
	req.NoError(err)

	b, err = New([]*grpc.ClientConn(nil), false, DefaultBroadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful broadcast threshold must be greater than 0")

	b, err = New(make([]*grpc.ClientConn, 1), false, DefaultBroadcastTimeout, 2)
	req.Nil(b)
	req.EqualError(err, "number of gateway connections (1) must be greater or equal than the successful broadcast threshold (2)")
}
