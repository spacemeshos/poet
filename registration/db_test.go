package registration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/poet/shared"
)

func TestInsertAndGetProof(t *testing.T) {
	tempdir := t.TempDir()
	db, err := newDatabase(tempdir, []byte{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	require.NoError(t, db.SaveProof(context.Background(), shared.NIP{Epoch: 1}, [][]byte{}))
	proof, err := db.GetProof(context.Background(), "1")
	require.NoError(t, err)
	require.Equal(t, "1", proof.RoundID)
}
