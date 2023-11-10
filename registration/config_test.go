package registration_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/registration"
)

func TestBase64EncDecode(t *testing.T) {
	enc := base64.StdEncoding.EncodeToString([]byte("hello"))
	b64 := registration.Base64Enc{}
	require.NoError(t, b64.UnmarshalFlag(enc))
	require.Equal(t, []byte("hello"), b64.Bytes())
}
