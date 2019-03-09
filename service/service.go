package service

import (
	"fmt"
	"time"
)

type Config struct {
	N                    uint          `long:"n" description:"PoET time parameter"`
	HashFunction         string        `long:"hashfunction" description:"PoET hash function"`
	RoundsDuration       time.Duration `long:"duration" description:"duration of the opening time for each round. If not specified, rounds duration will be determined by its previous round end of PoET execution"`
	InitialRoundDuration time.Duration `long:"initialduration" description:"duration of the opening time for the initial round. if rounds duration isn't specified, this param is necessary"`
	ExecuteEmpty         bool          `long:"empty" description:"whether to execution empty rounds, without any submitted commitments"`
}

type Service struct {
	cfg         *Config
	rounds      map[uint]*round
	activeRound *round
	prevRound   *round
}

type submitResponse struct {
	roundId uint
}

func NewService(cfg *Config) (*Service, error) {
	s := new(Service)
	s.cfg = cfg
	s.rounds = make(map[uint]*round)

	roundId := uint(0)
	s.activeRound = s.newRound(roundId)
	log.Infof("round %v opened", roundId)

	go func() {
		for {
			// Proceed either on previous round end of execution
			// or on the rounds ticker.
			select {
			case <-s.prevRoundExecuted():
			case <-s.roundsTicker():
			}

			if len(s.activeRound.commitments) == 0 && !s.cfg.ExecuteEmpty {
				continue
			}

			s.prevRound = s.activeRound

			roundId++
			s.activeRound = s.newRound(roundId)
			log.Infof("round %v opened", roundId)

			// Close previous round and execute it.
			go func() {
				r := s.prevRound
				err := r.close()
				if err != nil {
					log.Error(err)
				}
				log.Infof("round %v closed, executing...", r.id)
				phi, err := r.execute()
				if err != nil {
					log.Error(err)
				}
				log.Infof("round %v executed, phi=%v", r.id, phi)
			}()
		}
	}()

	return s, nil
}

func (s *Service) SubmitCommitment(c []byte) (*submitResponse, error) {
	r := s.activeRound
	err := r.submitCommitment(c)
	if err != nil {
		return nil, err
	}

	res := new(submitResponse)
	res.roundId = r.id
	return res, nil
}

func (s *Service) MembershipProof(roundId uint, c []byte) ([][]byte, error) {
	r, err := s.Round(roundId)
	if err != nil {
		return nil, err
	}

	proof, err := r.membershipProof(c)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (s *Service) Round(id uint) (*round, error) {
	r := s.rounds[id]
	if r == nil {
		return nil, fmt.Errorf("round %v not found", id)
	}

	return r, nil
}

func (s *Service) newRound(id uint) *round {
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
