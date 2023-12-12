package registration

import (
	"bytes"
	"context"
	"fmt"

	xdr "github.com/nullstyle/go-xdr/xdr3"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/spacemeshos/poet/shared"
)

var ErrNotFound = leveldb.ErrNotFound

type proof struct {
	// The actual proof.
	shared.MerkleProof

	// Members is the ordered list of miners challenges which are included
	// in the proof (by using the list hash digest as the proof generation input (the statement)).
	Members [][]byte

	// NumLeaves is the width of the proof-generation tree.
	NumLeaves uint64
}

type proofData struct {
	Proof         proof
	ServicePubKey []byte
	RoundID       string
}

type database struct {
	db     *leveldb.DB
	pubkey []byte
}

func newDatabase(dbPath string, pubkey []byte) (*database, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database @ %s: %w", dbPath, err)
	}

	return &database{db, pubkey}, nil
}

func (db *database) Close() error {
	return db.db.Close()
}

func (db *database) SaveProof(ctx context.Context, nip shared.NIP, members [][]byte) error {
	proof := proofData{
		proof{
			nip.MerkleProof,
			members,
			nip.Leaves,
		},
		db.pubkey,
		epochToRoundId(nip.Epoch),
	}
	serialized, err := serializeProof(proof)
	if err != nil {
		return fmt.Errorf("failed serializing proof: %w", err)
	}
	if err := db.db.Put([]byte(proof.RoundID), serialized, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("storing proof in DB: %w", err)
	}
	return nil
}

func (db *database) GetProof(ctx context.Context, roundID string) (*proofData, error) {
	data, err := db.db.Get([]byte(roundID), nil)
	if err != nil {
		return nil, fmt.Errorf("get proof for %s from DB: %w", roundID, err)
	}

	proof := &proofData{}
	_, err = xdr.Unmarshal(bytes.NewReader(data), proof)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize: %v", err)
	}
	return proof, nil
}

func serializeProof(proof proofData) ([]byte, error) {
	var dataBuf bytes.Buffer
	_, err := xdr.Marshal(&dataBuf, proof)
	if err != nil {
		return nil, fmt.Errorf("serialization failure: %v", err)
	}

	return dataBuf.Bytes(), nil
}
