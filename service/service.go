package service

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	xdr "github.com/nullstyle/go-xdr/xdr3"
	mshared "github.com/spacemeshos/merkle-tree/shared"
	"github.com/spacemeshos/poet/broadcaster"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/crypto/ed25519"
)

type Config struct {
	Genesis                  string        `long:"genesis-time" description:"Genesis timestamp"`
	EpochDuration            time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift               time.Duration `long:"phase-shift"`
	CycleGap                 time.Duration `long:"cycle-gap"`
	MemoryLayers             uint          `long:"memory" description:"Number of top Merkle tree layers to cache in-memory"`
	ExecuteEmpty             bool          `long:"empty" description:"whether to execute empty rounds, without any submitted challenges"`
	NoRecovery               bool          `long:"norecovery" description:"whether to disable a potential recovery procedure"`
	Reset                    bool          `long:"reset" description:"whether to reset the service state by deleting the datadir"`
	GatewayAddresses         []string      `long:"gateway" description:"list of Spacemesh gateway nodes RPC listeners (host:port) for broadcasting of proofs"`
	DisableBroadcast         bool          `long:"disablebroadcast" description:"whether to disable broadcasting of proofs"`
	ConnAcksThreshold        uint          `long:"conn-acks" description:"number of required successful connections to Spacemesh gateway nodes"`
	BroadcastAcksThreshold   uint          `long:"broadcast-acks" description:"number of required successful broadcasts via Spacemesh gateway nodes"`
	BroadcastNumRetries      uint          `long:"broadcast-num-retries" description:"number of broadcast retries"`
	BroadcastRetriesInterval time.Duration `long:"broadcast-retries-interval" description:"duration interval between broadcast retries"`
}

// estimatedLeavesPerSecond is used to computed estimated height of the proving tree
// in the epoch, which is used for cache estimation
const estimatedLeavesPerSecond = 1 << 17

const serviceStateFileBaseName = "state.bin"

type serviceState struct {
	PrivKey []byte
}

// Service orchestrates rounds functionality; each responsible for accepting challenges,
// generating a proof from their hash digest, and broadcasting the result to the Spacemesh network.
type Service struct {
	cfg            *Config
	datadir        string
	genesis        time.Time
	minMemoryLayer uint
	started        int32

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	openRound *round

	// executingRounds are the rounds which are currently executing, hence generating a proof.
	executingRounds map[string]*round

	prevRound *round

	PubKey      ed25519.PublicKey
	privKey     ed25519.PrivateKey
	broadcaster Broadcaster

	errChan chan error
	sig     *signal.Signal
	sync.Mutex
}

type InfoResponse struct {
	OpenRoundID        string
	ExecutingRoundsIds []string
}

type PoetProof struct {
	N         uint
	Statement []byte
	Proof     *shared.MerkleProof
}

var (
	ErrNotStarted     = errors.New("service not started")
	ErrAlreadyStarted = errors.New("already started")
)

type Broadcaster interface {
	BroadcastProof(msg []byte, roundID string, members [][]byte) error
}

type GossipPoetProof struct {
	// The actual proof.
	shared.MerkleProof

	// Members is the ordered list of miners challenges which are included
	// in the proof (by using the list hash digest as the proof generation input (the statement)).
	Members [][]byte

	// NumLeaves is the width of the proof-generation tree.
	NumLeaves uint64
}

type PoetProofMessage struct {
	GossipPoetProof
	ServicePubKey []byte
	RoundID       string
	Signature     []byte
}

func NewService(sig *signal.Signal, cfg *Config, datadir string) (*Service, error) {
	s := new(Service)
	s.cfg = cfg
	genesis, err := time.Parse(time.RFC3339, cfg.Genesis)
	if err != nil {
		return nil, err
	}
	minMemoryLayer := int(mshared.RootHeightFromWidth(
		uint64(cfg.EpochDuration.Seconds()*estimatedLeavesPerSecond),
	)) - int(cfg.MemoryLayers)
	if minMemoryLayer < prover.LowestMerkleMinMemoryLayer {
		minMemoryLayer = prover.LowestMerkleMinMemoryLayer
	}
	log.Info("starting poet service. min memory layer: %v. genesis: %s", minMemoryLayer, cfg.Genesis)

	s.minMemoryLayer = uint(minMemoryLayer)
	s.genesis = genesis
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
		PrivKey: priv,
	}
}

func (s *Service) saveState() error {
	filename := filepath.Join(s.datadir, serviceStateFileBaseName)
	v := &serviceState{
		PrivKey: s.privKey,
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

func (s *Service) Start(b Broadcaster) error {
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return ErrAlreadyStarted
	}

	s.SetBroadcaster(b)

	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else if err := s.Recover(); err != nil {
		return fmt.Errorf("failed to recover: %v", err)
	}
	now := time.Now()
	epoch := time.Duration(0)
	if d := now.Sub(s.genesis); d > 0 {
		epoch = d / s.cfg.EpochDuration
	}
	if s.openRound == nil {
		s.openRound = s.newRound(uint32(epoch))
		log.Info("Round %v opened", s.openRound.ID)
	}

	go func() {
		timer := time.NewTimer(0)
		<-timer.C
		defer timer.Stop()
		for {
			start := s.genesis.
				Add(s.cfg.EpochDuration * time.Duration(s.openRound.Epoch())).
				Add(s.cfg.PhaseShift)
			if d := start.Sub(time.Now()); d > 0 {
				log.Info("Round %v waiting for execution to start for %v",
					s.openRound.ID, d)
				timer.Reset(d)
				select {
				case <-timer.C:
				case <-s.sig.ShutdownRequestedChan:
					log.Info("Shutdown requested, service shutting down")
					s.openRound = nil
					return
				}
			}

			if s.openRound.isEmpty() && !s.cfg.ExecuteEmpty {
				continue
			}

			s.prevRound = s.openRound
			open := time.Now()
			// TODO(dshulyak) some time is wasted here to persist data on disk
			// on my computer ~20ms
			s.openRound = s.newRound(s.prevRound.Epoch() + 1)
			log.Info("Round %v opened. took %v", s.openRound.ID, time.Since(open))

			go func(r *round) {
				if err := s.executeRound(r); err != nil {
					s.asyncError(fmt.Errorf("round %v execution error: %v", r.ID, err))
					return
				}
				broadcastProof(s, r, r.execution, s.broadcaster)
			}(s.prevRound)
		}
	}()

	return nil
}

func (s *Service) Started() bool {
	return atomic.LoadInt32(&s.started) == 1
}

func (s *Service) Recover() error {
	entries, err := ioutil.ReadDir(s.datadir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}
		r := newRound(s.sig, s.cfg, s.datadir, uint32(epoch))
		state, err := r.state()
		if err != nil {
			return fmt.Errorf("invalid round state: %v", err)
		}

		if state.isExecuted() {
			log.Info("Recovery: found round %v in executed state. broadcasting...", r.ID)
			go broadcastProof(s, r, state.Execution, s.broadcaster)
			continue
		}

		if state.isOpen() {
			log.Info("Recovery: found round %v in open state.", r.ID)
			if err := r.open(); err != nil {
				return fmt.Errorf("failed to open round: %v", err)
			}

			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			s.openRound = r
			continue
		}

		log.Info("Recovery: found round %v in executing state. recovering execution...", r.ID)
		s.Lock()
		s.executingRounds[r.ID] = r
		s.Unlock()
		go func(r *round, rs *roundState) {
			defer func() {
				s.Lock()
				delete(s.executingRounds, r.ID)
				s.Unlock()
			}()

			end := s.genesis.
				Add(s.cfg.EpochDuration * time.Duration(r.Epoch()+1)).
				Add(s.cfg.PhaseShift).
				Add(-s.cfg.CycleGap)

			if err = r.recoverExecution(rs.Execution, end); err != nil {
				s.asyncError(fmt.Errorf("recovery: round %v execution failure: %v", r.ID, err))
				return
			}

			log.Info("Recovery: round %v execution ended, phi=%x", r.ID, r.execution.NIP.Root)
			broadcastProof(s, r, r.execution, s.broadcaster)
		}(r, state)
	}

	return nil
}

func (s *Service) SetBroadcaster(b Broadcaster) {
	initial := s.broadcaster == nil
	s.broadcaster = b

	if !initial {
		log.Info("Service broadcaster updated")
	}

}

func (s *Service) executeRound(r *round) error {
	s.Lock()
	s.executingRounds[r.ID] = r
	s.Unlock()

	defer func() {
		s.Lock()
		delete(s.executingRounds, r.ID)
		s.Unlock()
	}()

	start := time.Now()
	end := s.genesis.
		Add(s.cfg.EpochDuration * time.Duration(r.Epoch()+1)).
		Add(s.cfg.PhaseShift).
		Add(-s.cfg.CycleGap)

	log.Info("Round %v executing until %v...", r.ID, end)

	if err := r.execute(end, uint(s.minMemoryLayer)); err != nil {
		return err
	}

	log.Info("Round %v execution ended, phi=%x, duration %v", r.ID, r.execution.NIP.Root, time.Since(start))

	return nil
}

func (s *Service) Submit(data []byte) (*round, error) {
	if !s.Started() {
		return nil, ErrNotStarted
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
		return nil, ErrNotStarted
	}

	res := new(InfoResponse)
	res.OpenRoundID = s.openRound.ID

	s.Lock()
	ids := make([]string, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	s.Unlock()
	res.ExecutingRoundsIds = ids

	return res, nil
}

func (s *Service) newRound(epoch uint32) *round {
	if err := s.saveState(); err != nil {
		panic(err)
	}
	r := newRound(s.sig, s.cfg, s.datadir, epoch)
	if err := r.open(); err != nil {
		panic(fmt.Errorf("failed to open round: %v", err))
	}
	return r
}

func (s *Service) asyncError(err error) {
	capitalized := strings.ToUpper(err.Error()[0:1]) + err.Error()[1:]
	log.Error(capitalized)

	s.errChan <- err
}

func broadcastProof(s *Service, r *round, execution *executionState, broadcaster Broadcaster) {
	msg, err := serializeProofMsg(s.PubKey, r.ID, execution)
	if err != nil {
		log.Error(err.Error())
		return
	}

	bindFunc := func() error { return broadcaster.BroadcastProof(msg, r.ID, r.execution.Members) }
	logger := func(msg string) { log.Error("Round %v: %v", r.ID, msg) }

	if err := shared.Retry(bindFunc, int(s.cfg.BroadcastNumRetries), s.cfg.BroadcastRetriesInterval, logger); err != nil {
		log.Error("Round %v proof broadcast failure: %v", r.ID, err)
		return
	}

	r.broadcasted()
}

func serializeProofMsg(servicePubKey []byte, roundID string, execution *executionState) ([]byte, error) {
	proofMessage := PoetProofMessage{
		GossipPoetProof: GossipPoetProof{
			MerkleProof: *execution.NIP,
			Members:     execution.Members,
			NumLeaves:   execution.NumLeaves,
		},
		ServicePubKey: servicePubKey,
		RoundID:       roundID,
		Signature:     nil,
	}

	var dataBuf bytes.Buffer
	if _, err := xdr.Marshal(&dataBuf, proofMessage); err != nil {
		return nil, fmt.Errorf("failed to marshal proof message for round %v: %v", roundID, err)
	}

	return dataBuf.Bytes(), nil
}
