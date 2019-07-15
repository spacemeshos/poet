package service

import (
	"bytes"
	"errors"
	"fmt"
	xdr "github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/smutil/log"
	"sync"
	"time"
)

type Config struct {
	N                    uint          `long:"n" description:"PoET time parameter"`
	RoundsDuration       time.Duration `long:"duration" description:"duration of the opening time for each round. If not specified, rounds duration will be determined by its previous round end of PoET execution"`
	InitialRoundDuration time.Duration `long:"initialduration" description:"duration of the opening time for the initial round. if rounds duration isn't specified, this param is necessary"`
	ExecuteEmpty         bool          `long:"empty" description:"whether to execution empty rounds, without any submitted challenges"`
}

type Service struct {
	cfg             *Config
	openRound       *round
	prevRound       *round
	executingRounds map[int]*round

	errChan chan error

	sync.Mutex
}

type InfoResponse struct {
	OpenRoundId        int
	ExecutingRoundsIds []int
}

type MembershipProof struct {
	Index int
	Root  []byte
	Proof [][]byte
}

type PoetProof struct {
	N          uint
	Commitment []byte
	Proof      *shared.MerkleProof
}

var (
	ErrRoundNotFound = errors.New("round not found")
)

type Broadcaster interface {
	BroadcastProof(msg []byte) error
}

type GossipPoetProof struct {
	shared.MerkleProof
	Members   [][]byte
	LeafCount uint64
}

const PoetIdLength = 32

type PoetProofMessage struct {
	GossipPoetProof
	PoetId    [PoetIdLength]byte
	RoundId   uint64
	Signature []byte
}

func NewService(cfg *Config) (*Service, error) {
	s := new(Service)
	s.cfg = cfg
	s.executingRounds = make(map[int]*round)
	s.errChan = make(chan error)
	s.openRound = s.newRound(1)
	log.Info("round %v opened", 1)

	return s, nil
}

func (s *Service) Start(broadcaster Broadcaster) {
	go func() {
		for {
			// Proceed either on previous round end of execution
			// or on the rounds ticker.
			select {
			case <-s.prevRoundExecuted():
			case <-s.roundsTicker():
			}

			if len(s.openRound.challenges) == 0 && !s.cfg.ExecuteEmpty {
				continue
			}

			s.prevRound = s.openRound

			s.openRound = s.newRound(s.openRound.Id + 1)
			log.Info("round %v opened", s.openRound.Id)

			// Close previous round and execute it.
			go func() {
				r := s.prevRound

				s.Lock()
				s.executingRounds[r.Id] = r
				s.Unlock()

				err := r.close()
				if err != nil {
					s.errChan <- err
					log.Error(err.Error())
				}
				log.Info("round %v closed, executing...", r.Id)
				err = r.execute()
				if err != nil {
					s.errChan <- err
					log.Error(err.Error())
				}

				go broadcastProof(r, broadcaster)

				s.Lock()
				delete(s.executingRounds, r.Id)
				s.Unlock()

				log.Info("round %v executed, phi=%x", r.Id, r.nip.Root)
			}()
		}
	}()
}

func (s *Service) Submit(data []byte) (*round, error) {
	r := s.openRound
	err := r.submit(data)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) Info() *InfoResponse {
	res := new(InfoResponse)
	res.OpenRoundId = s.openRound.Id

	ids := make([]int, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	res.ExecutingRoundsIds = ids

	return res
}

func (s *Service) newRound(id int) *round {
	return newRound(s.cfg, id)
}

func (s *Service) prevRoundExecuted() <-chan struct{} {
	if s.prevRound != nil {
		return s.prevRound.executedChan
	} else {
		// If there's no previous round, then it's the initial round,
		// So simulate the previous round end of execution with the
		// initial round duration config.
		executedChan := make(chan struct{})
		go func() {
			<-time.After(s.cfg.InitialRoundDuration)
			close(executedChan)
		}()
		return executedChan
	}
}

var dummyChan = make(chan time.Time)

func (s *Service) roundsTicker() <-chan time.Time {
	if s.cfg.RoundsDuration > 0 {
		return time.After(s.cfg.RoundsDuration)
	} else {
		return dummyChan
	}
}

func broadcastProof(r *round, broadcaster Broadcaster) {
	if msg, err := serializeProofMsg(r); err != nil {
		log.Error(err.Error())
	} else if err := broadcaster.BroadcastProof(msg); err != nil {
		log.Error("failed to broadcast poet message for round %v: %v", r.Id, err)
	}
}

func serializeProofMsg(r *round) ([]byte, error) {
	poetProof, err := r.proof(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get poet proof for round %d: %v", r.Id, err)
	}
	proofMessage := PoetProofMessage{
		GossipPoetProof: GossipPoetProof{
			MerkleProof: *r.nip,
			Members:     r.challenges,
			LeafCount:   uint64(1) << poetProof.N,
		},
		PoetId:    [32]byte{},
		RoundId:   uint64(r.Id),
		Signature: nil,
	}
	var dataBuf bytes.Buffer
	_, err = xdr.Marshal(&dataBuf, proofMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof message for round %d: %v", r.Id, err)
	}
	return dataBuf.Bytes(), nil
}
