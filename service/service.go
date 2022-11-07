package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
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
	"github.com/spacemeshos/post/verifying"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/spacemeshos/poet/gateway/activation"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/signing"
	"github.com/spacemeshos/poet/types"
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

const AtxCacheSize = 1024

type serviceState struct {
	PrivKey []byte
}

// Service orchestrates rounds functionality; each responsible for accepting challenges,
// generating a proof from their hash digest, and broadcasting the result to the Spacemesh network.
//
// Service is single-use, meaning it can be started with `Start()` and then stopped with `Shutdown()`
// but it cannot be restarted. A new instance of `Service` must be created.
type Service struct {
	runningGroup errgroup.Group
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
	executingRounds      map[string]*round
	executingRoundsMutex sync.RWMutex

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey

	broadcaster atomic.Value // holds Broadcaster interface
	atxProvider atomic.Value // holds types.AtxProvider

	postConfig atomic.Pointer[types.PostConfig]

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
	var executingRounds errgroup.Group
	defer executingRounds.Wait()

	for {
		s.openRoundMutex.RLock()
		epoch := s.openRound.Epoch()
		s.openRoundMutex.RUnlock()

		start := s.genesis.Add(s.cfg.EpochDuration * time.Duration(epoch)).Add(s.cfg.PhaseShift)
		waitTime := time.Until(start)
		timer := time.After(waitTime)
		if waitTime > 0 {
			log.Info("Round %v waiting for execution to start for %v", s.openRoundID(), waitTime)
		}
		select {
		case <-timer:
		case <-ctx.Done():
			log.Info("service shutting down")
			s.openRoundMutex.Lock()
			s.openRound = nil
			s.openRoundMutex.Unlock()
			return
		}

		s.openRoundMutex.Lock()
		if s.openRound.isEmpty() && !s.cfg.ExecuteEmpty {
			// FIXME Poet enters a busy loop
			// See https://github.com/spacemeshos/poet/issues/142
			log.With().Info("Not executing an empty round", log.String("ID", s.openRound.ID))
			s.openRoundMutex.Unlock()
			continue
		}

		prevRound := s.openRound
		s.newRound(ctx, prevRound.Epoch()+1)
		s.openRoundMutex.Unlock()

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

func (s *Service) Start(b Broadcaster, atxProvider types.AtxProvider, postConfig *types.PostConfig) error {
	s.Lock()
	defer s.Unlock()
	if s.Started() {
		return ErrAlreadyStarted
	}

	// Create a context for the running Service.
	// This context will be canceled when Service is stopped.
	ctx, stop := context.WithCancel(context.Background())
	s.stop = stop

	s.SetBroadcaster(b)
	s.SetAtxProvider(atxProvider)
	s.SetPostConfig(postConfig)

	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else if err := s.recover(ctx); err != nil {
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
	}
	s.openRoundMutex.Unlock()

	s.runningGroup.Go(func() error {
		s.loop(ctx)
		return nil
	})
	s.started.Store(true)
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

func (s *Service) recover(ctx context.Context) error {
	log.With().Info("Recovering service state", log.String("datadir", s.datadir))
	entries, err := os.ReadDir(s.datadir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		log.Info("Recovering entry %s", entry.Name())
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
		s.executingRoundsMutex.Lock()
		s.executingRounds[r.ID] = r
		s.executingRoundsMutex.Unlock()
		s.runningGroup.Go(func() error {
			r, rs := r, state
			defer func() {
				s.executingRoundsMutex.Lock()
				delete(s.executingRounds, r.ID)
				s.executingRoundsMutex.Unlock()
			}()

			end := s.genesis.
				Add(s.cfg.EpochDuration * time.Duration(r.Epoch()+1)).
				Add(s.cfg.PhaseShift).
				Add(-s.cfg.CycleGap)

			if err = r.recoverExecution(ctx, rs.Execution, end); err != nil {
				s.asyncError(fmt.Errorf("recovery: round %v execution failure: %v", r.ID, err))
				return nil
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

func (s *Service) SetAtxProvider(provider types.AtxProvider) {
	s.atxProvider.Store(provider)
}

func (s *Service) SetPostConfig(config *types.PostConfig) {
	s.postConfig.Store(config)
}

func (s *Service) executeRound(ctx context.Context, r *round) error {
	s.executingRoundsMutex.Lock()
	s.executingRounds[r.ID] = r
	s.executingRoundsMutex.Unlock()

	defer func() {
		s.executingRoundsMutex.Lock()
		delete(s.executingRounds, r.ID)
		s.executingRoundsMutex.Unlock()
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

// Temporarily support both the old and new challenge submission API.
// TODO(brozansk) remove support for data []byte after go-spacemesh is updated to
// use the new API.
func (s *Service) Submit(ctx context.Context, challenge []byte, signedChallenge signing.Signed[shared.Challenge]) (*round, []byte, error) {
	if !s.Started() {
		return nil, nil, ErrNotStarted
	}

	// TODO(brozansk) Remove support for `challenge []byte` eventually
	if signedChallenge != nil {
		log.Debug("Using the new challenge submission API")
		// SAFETY: it will never panic as `s.atxProvider` is set in Start
		atxProvider := s.atxProvider.Load().(types.AtxProvider)
		if err := validateChallenge(ctx, signedChallenge, atxProvider, s.postConfig.Load(), verifying.Verify); err != nil {
			return nil, nil, err
		}
		// TODO(brozansk) calculate sequence number
		// TODO(brozansk) calculate hash

		s.openRoundMutex.Lock()
		r := s.openRound
		err := r.submit(signedChallenge.PubKey(), challenge)
		s.openRoundMutex.Unlock()
		if err != nil {
			return nil, nil, err
		}
		return r, []byte("TODO(brozansk) calculate hash"), nil
	} else {
		log.Debug("Using the old challenge submission API")
		s.openRoundMutex.Lock()
		r := s.openRound
		err := r.submit(challenge, challenge)
		s.openRoundMutex.Unlock()
		if err != nil {
			return nil, nil, err
		}
		return r, challenge, nil
	}
}

func (s *Service) Info() (*InfoResponse, error) {
	if !s.Started() {
		return nil, ErrNotStarted
	}

	s.executingRoundsMutex.RLock()
	ids := make([]string, 0, len(s.executingRounds))
	for id := range s.executingRounds {
		ids = append(ids, id)
	}
	s.executingRoundsMutex.RUnlock()

	return &InfoResponse{
		OpenRoundID:        s.openRoundID(),
		ExecutingRoundsIds: ids,
	}, nil
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
	log.With().Info("Round opened", log.String("ID", s.openRound.ID))
}

func (s *Service) openRoundID() string {
	s.openRoundMutex.RLock()
	defer s.openRoundMutex.RUnlock()
	return s.openRound.ID
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

// CreateAtxProvider creates ATX provider connected to provided gateways.
// The provider caches ATXes and round-robins between gateways if
// fetching new ATX failed.
func CreateAtxProvider(gateways []*grpc.ClientConn) (types.AtxProvider, error) {
	if len(gateways) == 0 {
		return nil, fmt.Errorf("need at least one gateway")
	}
	activationSvcClients := make([]types.AtxProvider, 0, len(gateways))
	for _, target := range gateways {
		client := activation.NewClient(target)
		activationSvcClients = append(activationSvcClients, client)
	}
	return activation.NewCachedAtxProvider(AtxCacheSize, activation.NewRoundRobinAtxProvider(activationSvcClients))
}
