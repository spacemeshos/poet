package service

import (
	"errors"
	"github.com/spacemeshos/poet/shared"
	"time"
)

type Config struct {
	N                    uint          `long:"n" description:"PoET time parameter"`
	HashFunction         string        `long:"hashfunction" description:"PoET hash function"`
	RoundsDuration       time.Duration `long:"duration" description:"duration of the opening time for each round. If not specified, rounds duration will be determined by its previous round end of PoET execution"`
	InitialRoundDuration time.Duration `long:"initialduration" description:"duration of the opening time for the initial round. if rounds duration isn't specified, this param is necessary"`
	ExecuteEmpty         bool          `long:"empty" description:"whether to execution empty rounds, without any submitted challenges"`
}

type Service struct {
	cfg             *Config
	openRound       *round
	prevRound       *round
	rounds          map[int]*round
	executingRounds map[int]*round
	executedRounds  map[int]*round

	errChan chan error
}

type InfoResponse struct {
	OpenRoundId        int
	ExecutingRoundsIds []int
	ExecutedRoundsIds  []int
}

type RoundInfoResponse struct {
	Opened          time.Time
	ExecuteStart    time.Time
	ExecuteEnd      time.Time
	ChallengesCount int
	MerkleRoot      []byte
	Nip             *shared.MerkleProof
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

func NewService(cfg *Config) (*Service, error) {
	s := new(Service)
	s.cfg = cfg
	s.rounds = make(map[int]*round)
	s.executingRounds = make(map[int]*round)
	s.executedRounds = make(map[int]*round)
	s.errChan = make(chan error)

	roundId := 1
	s.openRound = s.newRound(roundId)
	log.Infof("round %v opened", roundId)

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

			roundId++
			s.openRound = s.newRound(roundId)
			log.Infof("round %v opened", roundId)

			// Close previous round and execute it.
			go func() {
				// TODO(moshababo): apply safe concurrency
				r := s.prevRound
				s.executingRounds[r.Id] = r

				err := r.close()
				if err != nil {
					s.errChan <- err
					log.Error(err)
				}
				log.Infof("round %v closed, executing...", r.Id)
				err = r.execute()
				if err != nil {
					s.errChan <- err
					log.Error(err)
				}

				delete(s.executingRounds, r.Id)
				s.executedRounds[r.Id] = r
				log.Infof("round %v executed, phi=%v", r.Id, r.nip.Root)
			}()
		}
	}()

	return s, nil
}

func (s *Service) Submit(data []byte) (*round, error) {
	r := s.openRound
	err := r.submit(data)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) MembershipProof(roundId int, challenge []byte, wait bool) (*MembershipProof, error) {
	r := s.rounds[roundId]
	if r == nil {
		return nil, ErrRoundNotFound
	}

	proof, err := r.membershipProof(challenge, wait)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (s *Service) Proof(roundId int, wait bool) (*PoetProof, error) {
	r := s.rounds[roundId]
	if r == nil {
		return nil, ErrRoundNotFound
	}

	proof, err := r.proof(wait)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (s *Service) RoundInfo(roundId int) (*RoundInfoResponse, error) {
	r := s.rounds[roundId]
	if r == nil {
		return nil, ErrRoundNotFound
	}

	res := new(RoundInfoResponse)
	res.Opened = r.opened
	res.ExecuteStart = r.executeStart
	res.ExecuteEnd = r.executeEnd
	res.ChallengesCount = len(r.challenges)
	res.MerkleRoot = r.merkleRoot
	res.Nip = r.nip

	return res, nil
}

func (s *Service) Info() *InfoResponse {
	res := new(InfoResponse)
	res.OpenRoundId = s.openRound.Id

	ids := make([]int, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	res.ExecutingRoundsIds = ids

	ids = make([]int, 0, len(s.executedRounds))
	for id := range s.executedRounds {
		ids = append(ids, id)
	}
	res.ExecutedRoundsIds = ids

	return res
}

func (s *Service) round(roundId int) (*round, error) {
	r := s.rounds[roundId]
	if r == nil {
		return nil, ErrRoundNotFound
	}

	return r, nil
}

func (s *Service) newRound(id int) *round {
	r := newRound(s.cfg, id)
	s.rounds[id] = r
	return r
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
