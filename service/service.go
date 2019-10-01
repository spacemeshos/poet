package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/crypto/ed25519"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	N                    uint          `long:"n" description:"PoET time parameter"`
	RoundsDuration       time.Duration `long:"duration" description:"duration of the opening time for each round. If not specified, rounds duration will be determined by its previous round end of PoET execution"`
	InitialRoundDuration time.Duration `long:"initialduration" description:"duration of the opening time for the initial round. if rounds duration isn't specified, this param is necessary"`
	ExecuteEmpty         bool          `long:"empty" description:"whether to execution empty rounds, without any submitted challenges"`
	NoRecovery           bool          `long:"norecovery" description:"whether to disable a potential recovery procedure"`
}

const serviceStateFileBaseName = "state.bin"

type serviceState struct {
	NextRoundId int
	PrivKey     []byte
}

type Service struct {
	cfg             *Config
	datadir         string
	openRound       *round
	prevRound       *round
	executingRounds map[string]*round
	nextRoundId     int

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey

	errChan chan error
	sig     *signal.Signal
	sync.Mutex
}

type InfoResponse struct {
	OpenRoundId        string
	ExecutingRoundsIds []string
}

type MembershipProof struct {
	Index int
	Root  []byte
	Proof [][]byte
}

type PoetProof struct {
	N         uint
	Statement []byte
	Proof     *shared.MerkleProof
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
	NumLeaves uint64
}

type PoetProofMessage struct {
	GossipPoetProof
	ServicePubKey []byte
	RoundId       string
	Signature     []byte
}

func NewService(sig *signal.Signal, cfg *Config, datadir string) (*Service, error) {
	s := new(Service)
	s.cfg = cfg
	s.datadir = datadir
	s.executingRounds = make(map[string]*round)
	s.errChan = make(chan error, 10)
	s.sig = sig

	state, err := s.state()
	if err != nil {
		if !strings.Contains(err.Error(), "file is missing") {
			return nil, err
		}
		state = s.initialState()
	}

	s.privKey = ed25519.NewKeyFromSeed(state.PrivKey[:32])
	s.PubKey = state.PrivKey[32:]
	s.nextRoundId = state.NextRoundId

	log.Info("Service public key: %x", s.PubKey)

	return s, nil
}

func (s *Service) initialState() *serviceState {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(fmt.Errorf("failed to generate key: %v", err))
	}

	return &serviceState{
		NextRoundId: 1,
		PrivKey:     priv,
	}
}

func (s *Service) saveState() error {
	filename := filepath.Join(s.datadir, serviceStateFileBaseName)
	v := &serviceState{
		NextRoundId: s.nextRoundId,
		PrivKey:     s.privKey,
	}

	return persist(filename, v)
}

func (s *Service) state() (*serviceState, error) {
	filename := filepath.Join(s.datadir, roundStateFileBaseName)
	v := &serviceState{}

	if err := load(filename, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (s *Service) Start(broadcaster Broadcaster) {
	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else if err := s.Recover(broadcaster); err != nil {
		log.Error("Failed to recover: %v", err)
		return
	}

	s.openRound = s.newRound()
	log.Info("Round %v opened", s.openRound.Id)

	go func() {
		for {
			// Proceed either on previous round end of execution
			// or on the rounds ticker.
			select {
			case <-s.prevRoundExecutionEnd():
			case <-s.roundsTicker():
			case <-s.sig.ShutdownRequestedChan:
				log.Info("Shutdown requested, service shutting down")
				s.openRound = nil
				return
			}

			if s.openRound.isEmpty() && !s.cfg.ExecuteEmpty {
				continue
			}

			s.prevRound = s.openRound
			s.openRound = s.newRound()
			log.Info("Round %v opened", s.openRound.Id)

			// Close previous round and execute it.
			go func() {
				r := s.prevRound
				if err := s.executeRound(r); err != nil {
					s.asyncError(fmt.Errorf("round %v execution error: %v", r.Id, err))
					return
				}

				go broadcastProof(s, r, broadcaster)
			}()
		}
	}()
}

func (s *Service) Recover(broadcaster Broadcaster) error {
	entries, err := ioutil.ReadDir(s.datadir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		datadir := filepath.Join(s.datadir, entry.Name())
		r := newRound(s.sig, s.cfg, datadir, entry.Name())

		state, err := r.state()
		if err != nil {
			return fmt.Errorf("invalid round state: %v", err)
		}

		go func() {
			s.Lock()
			s.executingRounds[r.Id] = r
			s.Unlock()
			defer func() {
				s.Lock()
				delete(s.executingRounds, r.Id)
				s.Unlock()
			}()

			var err error
			if state.isOpen() {
				log.Info("Recovery: found round %v in open state. executing...", r.Id)
				err = r.execute()
			} else {
				log.Info("Recovery: found round %v in executing state. recovering execution...", r.Id)
				err = r.recoverExecution(state.Execution)
			}

			if err != nil {
				s.asyncError(fmt.Errorf("recovery: round %v execution failure: %v", r.Id, err))
				return
			}

			log.Info("Recovery: round %v execution ended, phi=%x", r.Id, r.execution.NIP.Root)
			broadcastProof(s, r, broadcaster)
		}()
	}

	return nil
}

func (s *Service) executeRound(r *round) error {
	s.Lock()
	s.executingRounds[r.Id] = r
	s.Unlock()

	defer func() {
		s.Lock()
		delete(s.executingRounds, r.Id)
		s.Unlock()
	}()

	log.Info("Round %v executing...", r.Id)

	if err := r.execute(); err != nil {
		return err
	}

	log.Info("Round %v execution ended, phi=%x", r.Id, r.execution.NIP.Root)

	return nil
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

	ids := make([]string, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	res.ExecutingRoundsIds = ids

	return res
}

func (s *Service) newRound() *round {
	roundId := fmt.Sprintf("%d", s.nextRoundId)
	s.nextRoundId++
	if err := s.saveState(); err != nil {
		panic(err)
	}

	datadir := filepath.Join(s.datadir, roundId)

	r := newRound(s.sig, s.cfg, datadir, roundId)
	if err := r.open(); err != nil {
		panic(fmt.Errorf("failed to open round: %v", err))
	}

	return r
}

func (s *Service) prevRoundExecutionEnd() <-chan struct{} {
	if s.prevRound != nil {
		return s.prevRound.executionEndedChan
	} else {
		// If there's no previous round, then it's the initial round,
		// So simulate the previous round end of execution with the
		// initial round duration config.
		executionEndChan := make(chan struct{})
		go func() {
			<-time.After(s.cfg.InitialRoundDuration)
			close(executionEndChan)
		}()
		return executionEndChan
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

func (s *Service) asyncError(err error) {
	capitalized := strings.ToUpper(err.Error()[0:1]) + err.Error()[1:]
	log.Error(capitalized)

	s.errChan <- err
}

func broadcastProof(s *Service, r *round, broadcaster Broadcaster) {
	if msg, err := serializeProofMsg(s, r); err != nil {
		log.Error(err.Error())
	} else if err := broadcaster.BroadcastProof(msg); err != nil {
		log.Error("failed to broadcast poet message for round %v: %v", r.Id, err)
	}
}

func serializeProofMsg(s *Service, r *round) ([]byte, error) {
	poetProof, err := r.proof(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get poet proof for round %v: %v", r.Id, err)
	}

	proofMessage := PoetProofMessage{
		GossipPoetProof: GossipPoetProof{
			MerkleProof: *r.execution.NIP,
			Members:     r.execution.Members,
			NumLeaves:   uint64(1) << poetProof.N,
		},
		ServicePubKey: s.PubKey,
		RoundId:       r.Id,
		Signature:     nil,
	}
	var dataBuf bytes.Buffer
	_, err = xdr.Marshal(&dataBuf, proofMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof message for round %v: %v", r.Id, err)
	}
	return dataBuf.Bytes(), nil
}
