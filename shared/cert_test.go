package shared

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDecodeCert(t *testing.T) {
	encoded, err := base64.StdEncoding.DecodeString("gAECAwQFBgcICQoLDA0ODxABAgMEBQYHCAkKCwwNDg8QAXaB58o=")
	require.NoError(t, err)

	cert, err := DecodeCert(encoded)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, cert.Pubkey)

	exp, err := time.Parse(time.RFC3339, "1996-12-19T16:39:57-08:00")
	require.NoError(t, err)
	require.Equal(t, exp.UTC(), *cert.Expiration)
}
