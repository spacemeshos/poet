package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spacemeshos/go-scale"
	mshared "github.com/spacemeshos/merkle-tree/shared"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
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
// in the epoch, which is used for cache estimation.
const estimatedLeavesPerSecond = 1 << 17

const serviceStateFileBaseName = "state.bin"

type serviceState struct {
	PrivKey []byte
}

// Service orchestrates rounds functionality; each responsible for accepting challenges,
// generating a proof from their hash digest, and broadcasting the result to the Spacemesh network.
type Service struct {
	runningGroup *errgroup.Group
	stop         context.CancelFunc

	cfg            *Config
	datadir        string
	genesis        time.Time
	minMemoryLayer uint
	started        atomic.Bool

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	// openRoundMutex guards openRound, any access to it must be protected by this mutex.
	openRound      *round
	openRoundMutex sync.RWMutex

	// executingRounds are the rounds which are currently executing, hence generating a proof.
	executingRounds map[string]*round

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
	// holds Broadcaster interface
	broadcaster atomic.Value

	errChan chan error
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

//go:generate scalegen -types PoetProofMessage,GossipPoetProof

type PoetProofMessage struct {
	GossipPoetProof
	ServicePubKey []byte
	RoundID       string
	Signature     []byte
}

func NewService(cfg *Config, datadir string) (*Service, error) {
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
	log.Info("creating poet service. min memory layer: %v. genesis: %s", minMemoryLayer, cfg.Genesis)

	if cfg.Reset {
		entries, err := os.ReadDir(datadir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(datadir, entry.Name())); err != nil {
				return nil, err
			}
		}
	}

	state, err := state(datadir)
	if err != nil {
		if !errors.Is(err, ErrFileIsMissing) {
			return nil, err
		}
		state = initialState()
	}

	privateKey := ed25519.NewKeyFromSeed(state.PrivKey[:32])
	s := &Service{
		cfg:             cfg,
		minMemoryLayer:  uint(minMemoryLayer),
		genesis:         genesis,
		datadir:         datadir,
		executingRounds: make(map[string]*round),
		errChan:         make(chan error, 10),
		privKey:         privateKey,
		PubKey:          privateKey.Public().(ed25519.PublicKey),
	}
	log.Info("Service public key: %x", s.PubKey)

	return s, nil
}

func (s *Service) loop(ctx context.Context) {
	executingRounds, _ := errgroup.WithContext(ctx)
	defer executingRounds.Wait()

	timer := time.NewTimer(0)
	<-timer.C
	defer timer.Stop()
	for {
		start := s.genesis.
			Add(s.cfg.EpochDuration * time.Duration(s.openRound.Epoch())).
			Add(s.cfg.PhaseShift)
		if d := time.Until(start); d > 0 {
			log.Info("Round %v waiting for execution to start for %v", s.openRoundID(), d)
			timer.Reset(d)
			select {
			case <-timer.C:
			case <-ctx.Done():
				log.Info("service shutting down")
				s.openRoundMutex.Lock()
				s.openRound = nil
				s.openRoundMutex.Unlock()
				return
			}
		}

		s.openRoundMutex.Lock()
		if s.openRound.isEmpty() && !s.cfg.ExecuteEmpty {
			s.openRoundMutex.Unlock()
			continue
		}

		prevRound := s.openRound
		open := time.Now()
		s.newRound(ctx, prevRound.Epoch()+1)
		s.openRoundMutex.Unlock()
		log.Info("Round %v opened. took %v", s.openRoundID(), time.Since(open))

		executingRounds.Go(func() error {
			round := prevRound
			if err := s.executeRound(ctx, round); err != nil {
				s.asyncError(fmt.Errorf("round %v execution error: %v", round.ID, err))
				return nil
			}
			broadcastProof(s, round, round.execution, s.getBroadcaster())
			return nil
		})
	}
}

func (s *Service) Start(b Broadcaster) error {
	s.Lock()
	if !s.started.CompareAndSwap(false, true) {
		s.Unlock()
		return ErrAlreadyStarted
	}

	// Create a context for the running Service.
	// This context will be canceled when Service is stopped.
	ctx, stop := context.WithCancel(context.Background())
	runningGroup, _ := errgroup.WithContext(ctx)
	s.stop = stop
	s.runningGroup = runningGroup
	s.Unlock()

	s.SetBroadcaster(b)

	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else if err := s.Recover(ctx); err != nil {
		return fmt.Errorf("failed to recover: %v", err)
	}
	now := time.Now()
	epoch := time.Duration(0)
	if d := now.Sub(s.genesis); d > 0 {
		epoch = d / s.cfg.EpochDuration
	}
	s.openRoundMutex.Lock()
	if s.openRound == nil {
		s.newRound(ctx, uint32(epoch))
		log.Info("Round %v opened", s.openRound.ID)
	}
	s.openRoundMutex.Unlock()

	s.runningGroup.Go(func() error {
		s.loop(ctx)
		return nil
	})
	return nil
}

// Shutdown gracefully stops running Service and waits
// for all processing to stop.
func (s *Service) Shutdown() error {
	log.Info("requested service shutdown")
	if !s.Started() {
		return ErrNotStarted
	}
	s.stop()
	err := s.runningGroup.Wait()
	s.started.Store(false)
	log.Info("service shutdown complete")
	return err
}

func (s *Service) Started() bool {
	return s.started.Load()
}

func (s *Service) Recover(ctx context.Context) error {
	entries, err := os.ReadDir(s.datadir)
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
		r := newRound(ctx, s.datadir, uint32(epoch))
		state, err := r.state()
		if err != nil {
			return fmt.Errorf("invalid round state: %v", err)
		}

		if state.isExecuted() {
			log.Info("Recovery: found round %v in executed state. broadcasting...", r.ID)
			go broadcastProof(s, r, state.Execution, s.getBroadcaster())
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
		s.runningGroup.Go(func() error {
			r, rs := r, state
			defer func() {
				s.Lock()
				delete(s.executingRounds, r.ID)
				s.Unlock()
			}()

			end := s.genesis.
				Add(s.cfg.EpochDuration * time.Duration(r.Epoch()+1)).
				Add(s.cfg.PhaseShift).
				Add(-s.cfg.CycleGap)

			if err = r.recoverExecution(ctx, rs.Execution, end); err != nil {
				s.asyncError(fmt.Errorf("recovery: round %v execution failure: %v", r.ID, err))
				return err
			}

			log.Info("Recovery: round %v execution ended, phi=%x", r.ID, r.execution.NIP.Root)
			broadcastProof(s, r, r.execution, s.getBroadcaster())
			return nil
		})
	}

	return nil
}

func (s *Service) getBroadcaster() Broadcaster {
	return s.broadcaster.Load().(Broadcaster)
}

func (s *Service) SetBroadcaster(b Broadcaster) {
	if s.broadcaster.Swap(b) != nil {
		log.Info("Service broadcaster updated")
	}
}

func (s *Service) executeRound(ctx context.Context, r *round) error {
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

	if err := r.execute(ctx, end, uint(s.minMemoryLayer)); err != nil {
		return err
	}

	log.Info("Round %v execution ended, phi=%x, duration %v", r.ID, r.execution.NIP.Root, time.Since(start))

	return nil
}

func (s *Service) Submit(data []byte) (*round, error) {
	if !s.Started() {
		return nil, ErrNotStarted
	}

	s.openRoundMutex.Lock()
	r := s.openRound
	err := r.submit(data)
	s.openRoundMutex.Unlock()
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
	if id := s.openRoundID(); id != nil {
		res.OpenRoundID = *id
	}

	s.Lock()
	ids := make([]string, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	s.Unlock()
	res.ExecutingRoundsIds = ids

	return res, nil
}

// newRound creates a new round with the given epoch. This method MUST be guarded by a write lock on openRoundMutex.
func (s *Service) newRound(ctx context.Context, epoch uint32) {
	if err := saveState(s.datadir, s.privKey); err != nil {
		panic(err)
	}
	r := newRound(ctx, s.datadir, epoch)
	if err := r.open(); err != nil {
		panic(fmt.Errorf("failed to open round: %v", err))
	}

	s.openRound = r
}

func (s *Service) openRoundID() *string {
	s.openRoundMutex.RLock()
	defer s.openRoundMutex.RUnlock()
	if s.openRound != nil {
		return &s.openRound.ID
	}
	return nil
}

func (s *Service) asyncError(err error) {
	log.Error(err.Error())
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
	if _, err := proofMessage.EncodeScale(scale.NewEncoder(&dataBuf)); err != nil {
		return nil, fmt.Errorf("failed to marshal proof message for round %v: %v", roundID, err)
	}

	return dataBuf.Bytes(), nil
}
