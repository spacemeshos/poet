package service

import (
	"bytes"
	"context"
	"fmt"

	xdr "github.com/nullstyle/go-xdr/xdr3"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
)

var ErrNotFound = leveldb.ErrNotFound

type ProofsDatabase struct {
	db     *leveldb.DB
	proofs <-chan proofMessage
}

func (db *ProofsDatabase) Get(ctx context.Context, roundID string) (*proofMessage, error) {
	data, err := db.db.Get([]byte(roundID), nil)
	if err != nil {
		return nil, fmt.Errorf("get proof for %s from DB: %w", roundID, err)
	}

	proof := &proofMessage{}
	_, err = xdr.Unmarshal(bytes.NewReader(data), proof)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize: %v", err)
	}
	return proof, nil
}

func NewProofsDatabase(dbPath string, proofs <-chan proofMessage) (*ProofsDatabase, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database @ %s: %w", dbPath, err)
	}

	return &ProofsDatabase{db, proofs}, nil
}

func (db *ProofsDatabase) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx).Named("proofs-db")
	for {
		select {
		case proofMsg := <-db.proofs:
			serialized, err := serializeProofMsg(proofMsg)
			proof := proofMsg.Proof
			if err != nil {
				return fmt.Errorf("failed serializing proof: %w", err)
			}
			if err := db.db.Put([]byte(proofMsg.RoundID), serialized, &opt.WriteOptions{Sync: true}); err != nil {
				logger.Error("failed storing proof in DB", zap.Error(err))
			} else {
				logger.Info("Proof saved in DB",
					zap.String("round", proofMsg.RoundID),
					zap.Int("members", len(proof.Members)),
					zap.Uint64("leaves", proof.NumLeaves))
			}
		case <-ctx.Done():
			logger.Info("shutting down proofs db")
			return db.db.Close()
		}
	}
}

func serializeProofMsg(proof proofMessage) ([]byte, error) {
	var dataBuf bytes.Buffer
	_, err := xdr.Marshal(&dataBuf, proof)
	if err != nil {
		return nil, fmt.Errorf("serialization failure: %v", err)
	}

	return dataBuf.Bytes(), nil
}
