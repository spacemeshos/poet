package broadcaster

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	req := require.New(t)

	connTimeout := DefaultConnTimeout
	broadcastTimeout := DefaultBroadcastTimeout

	b, err := New([]string(nil), true, connTimeout, 0, broadcastTimeout, 0)
	req.NotNil(b)
	req.NoError(err)

	b, err = New([]string(nil), false, connTimeout, 0, broadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses must be greater than 0")

	b, err = New(make([]string, 0), false, connTimeout, 0, broadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses must be greater than 0")

	b, err = New(make([]string, 1), false, connTimeout, 0, broadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful connections threshold must be greater than 0")

	b, err = New(make([]string, 1), false, connTimeout, 1, broadcastTimeout, 0)
	req.Nil(b)
	req.EqualError(err, "successful broadcast threshold must be greater than 0")

	b, err = New(make([]string, 1), false, connTimeout, 2, broadcastTimeout, 1)
	req.Nil(b)
	req.EqualError(err, "number of gateway addresses (1) must be greater than the successful connections threshold (2)")

	b, err = New(make([]string, 1), false, connTimeout, 1, broadcastTimeout, 2)
	req.Nil(b)
	req.EqualError(err, "the successful connections threshold (1) must be greater than the successful broadcast threshold (2)")

	b, err = New([]string{"666"}, false, 0, 1, 0, 1)
	req.Nil(b)
	req.EqualError(err, "failed to connect to Spacemesh gateway node at \"666\": failed to connect to rpc server: context deadline exceeded")

	b, err = New([]string{"666", "667"}, false, 0, 1, 0, 1)
	req.Nil(b)
	req.EqualError(err, "failed to connect to Spacemesh gateway node at \"666\": failed to connect to rpc server: context deadline exceeded | failed to connect to Spacemesh gateway node at \"667\": failed to connect to rpc server: context deadline exceeded")
}
