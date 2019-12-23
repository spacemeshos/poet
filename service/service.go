package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/poet/broadcaster"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/crypto/ed25519"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	N                      uint          `long:"n" description:"PoET time parameter"`
	MemoryLayers           uint          `long:"memory" description:"Number of top Merkle tree layers to cache in-memory"`
	RoundsDuration         time.Duration `long:"duration" description:"duration of the opening time for each round. If not specified, rounds duration will be determined by its previous round end of PoET execution"`
	InitialRoundDuration   time.Duration `long:"initialduration" description:"duration of the opening time for the initial round. if rounds duration isn't specified, this param is necessary"`
	ExecuteEmpty           bool          `long:"empty" description:"whether to execute empty rounds, without any submitted challenges"`
	NoRecovery             bool          `long:"norecovery" description:"whether to disable a potential recovery procedure"`
	Reset                  bool          `long:"reset" description:"whether to reset the service state by deleting the datadir"`
	GatewayAddresses       []string      `long:"gateway" description:"list of Spacemesh gateway nodes RPC listeners (host:port) for broadcasting of proofs"`
	DisableBroadcast       bool          `long:"disablebroadcast" description:"whether to disable broadcasting of proofs"`
	ConnAcksThreshold      uint          `long:"conn-acks" description:"number of required successful connections to Spacemesh gateway nodes"`
	BroadcastAcksThreshold uint          `long:"broadcast-acks" description:"number of required successful broadcasts via Spacemesh gateway nodes"`
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

	started int32
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
	BroadcastProof(msg []byte, roundId string, members [][]byte) error
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

	if cfg.Reset {
		entries, err := ioutil.ReadDir(datadir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(s.datadir, entry.Name())); err != nil {
				return nil, err
			}
		}
	}

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

	if len(cfg.GatewayAddresses) > 0 || cfg.DisableBroadcast {
		b, err := broadcaster.New(
			cfg.GatewayAddresses,
			cfg.DisableBroadcast,
			broadcaster.DefaultConnTimeout,
			cfg.ConnAcksThreshold,
			broadcaster.DefaultBroadcastTimeout,
			cfg.BroadcastAcksThreshold,
		)
		if err != nil {
			return nil, err
		}

		if err := s.Start(b); err != nil {
			return nil, fmt.Errorf("failed to start service: %v", err)
		}
	} else {
		log.Info("Service not starting, waiting for start request")
	}

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

func (s *Service) Start(broadcaster Broadcaster) error {
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return errors.New("already started")
	}

	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else if err := s.Recover(broadcaster); err != nil {
		return fmt.Errorf("failed to recover: %v", err)
	}

	if s.openRound == nil {
		s.openRound = s.newRound()
		log.Info("Round %v opened", s.openRound.Id)
	}

	go func() {
		for {
			select {
			case <-s.openRoundClosure():
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

				broadcastProof(s, r, r.execution, broadcaster)
			}()
		}
	}()

	return nil
}

func (s *Service) Started() bool {
	return atomic.LoadInt32(&s.started) == 1
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

		if state.isOpen() {
			log.Info("Recovery: found round %v in open state", r.Id)

			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			s.openRound = r
			continue
		}

		if state.isExecuted() {
			log.Info("Recovery: found round %v in executed state. broadcasting...", r.Id)
			go broadcastProof(s, r, state.Execution, broadcaster)
			continue
		}

		log.Info("Recovery: found round %v in executing state. recovering execution...", r.Id)

		// Keep the last executing round as prevRound for potentially affecting
		// the closure of the current open round (see openRoundClosure()).
		s.prevRound = r

		go func() {
			s.Lock()
			s.executingRounds[r.Id] = r
			s.Unlock()
			defer func() {
				s.Lock()
				delete(s.executingRounds, r.Id)
				s.Unlock()
			}()

			if err = r.recoverExecution(state.Execution); err != nil {
				s.asyncError(fmt.Errorf("recovery: round %v execution failure: %v", r.Id, err))
				return
			}

			log.Info("Recovery: round %v execution ended, phi=%x", r.Id, r.execution.NIP.Root)
			broadcastProof(s, r, r.execution, broadcaster)
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
	if !s.Started() {
		return nil, errors.New("service not started")
	}

	r := s.openRound
	err := r.submit(data)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) Info() (*InfoResponse, error) {
	if !s.Started() {
		return nil, errors.New("service not started")
	}

	res := new(InfoResponse)
	res.OpenRoundId = s.openRound.Id

	ids := make([]string, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	res.ExecutingRoundsIds = ids

	return res, nil
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

// openRoundClosure returns a channel used to notify the closure of the current open round.
func (s *Service) openRoundClosure() <-chan struct{} {
	c := make(chan struct{})

	// If rounds duration was specified, use it to notify the closure.
	// If the open round was recovered, include the time period from when it was originally opened.
	if s.cfg.RoundsDuration > 0 {
		var offset time.Duration
		if s.openRound.stateCache != nil {
			offset = time.Since(s.openRound.stateCache.Opened)
		}
		go func() {
			<-time.After(s.cfg.RoundsDuration - offset)
			close(c)
		}()
		return c
	}

	// If it's not the initial round, use the previous round end of execution to notify the closure.
	if s.prevRound != nil {
		return s.prevRound.executionEndedChan
	}

	// Use the initial duration config to notify the closure.
	// If the open round was recovered, include the time period from when it was originally opened.
	var offset time.Duration
	if s.openRound.stateCache != nil {
		offset = time.Since(s.openRound.stateCache.Opened)
	}
	go func() {
		<-time.After(s.cfg.InitialRoundDuration - offset)
		close(c)
	}()
	return c
}

func (s *Service) asyncError(err error) {
	capitalized := strings.ToUpper(err.Error()[0:1]) + err.Error()[1:]
	log.Error(capitalized)

	s.errChan <- err
}

func broadcastProof(s *Service, r *round, execution *executionState, broadcaster Broadcaster) {
	msg, err := serializeProofMsg(s.PubKey, r.Id, execution)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err := broadcaster.BroadcastProof(msg, r.Id, r.execution.Members); err != nil {
		log.Error("Round %v proof broadcast failure: %v", r.Id, err)
		return
	}

	r.broadcasted()
}

func serializeProofMsg(servicePubKey []byte, roundId string, execution *executionState) ([]byte, error) {
	proofMessage := PoetProofMessage{
		GossipPoetProof: GossipPoetProof{
			MerkleProof: *execution.NIP,
			Members:     execution.Members,
			NumLeaves:   execution.NumLeaves,
		},
		ServicePubKey: servicePubKey,
		RoundId:       roundId,
		Signature:     nil,
	}

	var dataBuf bytes.Buffer
	if _, err := xdr.Marshal(&dataBuf, proofMessage); err != nil {
		return nil, fmt.Errorf("failed to marshal proof message for round %v: %v", roundId, err)
	}

	return dataBuf.Bytes(), nil
}
