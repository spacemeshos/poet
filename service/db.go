package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spacemeshos/go-scale"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
)

type dataType byte

const (
	typeProof dataType = iota
	typeRound
)

var ErrNotFound = leveldb.ErrNotFound

type ServiceDatabase struct {
	db     *leveldb.DB
	proofs <-chan shared.ProofMessage
	rounds <-chan RoundData
}

func makeKey(t dataType, k []byte) []byte {
	key := make([]byte, len(k)+1)
	key = append(key, byte(t))
	return append(key, k...)
}

func (db *ServiceDatabase) GetProof(ctx context.Context, roundID string) (*shared.ProofMessage, error) {
	data, err := db.db.Get(makeKey(typeProof, []byte(roundID)), nil)
	if err != nil {
		return nil, fmt.Errorf("getting proof for round %s from DB: %w", roundID, err)
	}

	proof := &shared.ProofMessage{}
	if _, err := proof.DecodeScale(scale.NewDecoder(bytes.NewReader(data))); err != nil {
		return nil, fmt.Errorf("deserializing proof for round %s: %w", roundID, err)
	}
	return proof, nil
}

func (db *ServiceDatabase) GetRoundMembers(ctx context.Context, roundID string) ([][]byte, error) {
	data, err := db.db.Get(makeKey(typeRound, []byte(roundID)), nil)
	if err != nil {
		return nil, fmt.Errorf("getting members for round %s from DB: %w", roundID, err)
	}

	if members, _, err := scale.DecodeSliceOfByteSlice(scale.NewDecoder(bytes.NewReader(data))); err != nil {
		return nil, fmt.Errorf("deserializing round %s members: %w", roundID, err)
	} else {
		return members, nil
	}
}

func NewServiceDatabase(
	dbPath string,
	proofs <-chan shared.ProofMessage,
	rounds <-chan RoundData,
) (*ServiceDatabase, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("opening database @ %s: %w", dbPath, err)
	}

	return &ServiceDatabase{db, proofs, rounds}, nil
}

func (db *ServiceDatabase) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx).Named("proofs-db")
	for {
		select {
		case proof := <-db.proofs:
			serialized, err := serializeProofMsg(proof)
			if err != nil {
				return fmt.Errorf("serializing proof: %w", err)
			}
			if err := db.db.Put(makeKey(typeProof, []byte(proof.RoundID)), serialized, &opt.WriteOptions{Sync: true}); err != nil {
				logger.Error("failed storing proof in DB", zap.String("round", proof.RoundID), zap.Error(err))
			} else {
				logger.Info("proof saved in DB",
					zap.String("round", proof.RoundID),
					zap.Uint64("leaves", proof.NumLeaves))
			}
		case round := <-db.rounds:
			var dataBuf bytes.Buffer
			if _, err := scale.EncodeSliceOfByteSlice(scale.NewEncoder(&dataBuf), round.Members); err != nil {
				return fmt.Errorf("serializing proof: %w", err)
			}

			if err := db.db.Put(makeKey(typeRound, []byte(round.ID)), dataBuf.Bytes(), &opt.WriteOptions{Sync: true}); err != nil {
				logger.Error("failed storing round members in DB", zap.String("round", round.ID), zap.Error(err))
			} else {
				logger.Info("round members saved in DB", zap.String("round", round.ID))
			}
		case <-ctx.Done():
			logger.Info("shutting down proofs db")
			return db.db.Close()
		}
	}
}

func serializeProofMsg(proof shared.ProofMessage) ([]byte, error) {
	var dataBuf bytes.Buffer
	if _, err := proof.EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, fmt.Errorf("marshaling proof message for round %v: %v", proof.RoundID, err)
	}

	return dataBuf.Bytes(), nil
}
