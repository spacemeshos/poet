package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/spacemeshos/poet/gateway/challenge_verifier"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
)

type Config struct {
	Genesis                  string        `long:"genesis-time" description:"Genesis timestamp"`
	EpochDuration            time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift               time.Duration `long:"phase-shift"`
	CycleGap                 time.Duration `long:"cycle-gap"`
	MemoryLayers             uint          `long:"memory" description:"Number of top Merkle tree layers to cache in-memory"`
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

const ChallengeVerifierCacheSize = 1024

// ServiceClient is an interface for interacting with the Service actor.
// It is created when the Service is started.
type ServiceClient struct {
	serviceStarted    *atomic.Bool
	command           chan<- Command
	challengeVerifier atomic.Value // holds challenge_verifier.Verifier
}

// Service orchestrates rounds functionality; each responsible for accepting challenges,
// generating a proof from their hash digest, and broadcasting the result to the Spacemesh network.
//
// Service is single-use, meaning it can be started with `Start()` and then stopped with `Shutdown()`
// but it cannot be restarted. A new instance of `Service` must be created.
type Service struct {
	commands <-chan Command
	ServiceClient

	runningGroup errgroup.Group
	stop         context.CancelFunc

	cfg            *Config
	datadir        string
	genesis        time.Time
	minMemoryLayer uint
	started        atomic.Bool

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	openRound       *round
	executingRounds map[string]struct{}

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey

	broadcaster Broadcaster

	sync.Mutex
}

// Command is a function that will be run in the main Service loop.
// Commands are run serially hence they don't require additional synchronization.
// The functions cannot block and should be kept short to not block the Service loop.
type Command func(*Service)

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
	ErrNotStarted                = errors.New("service not started")
	ErrAlreadyStarted            = errors.New("already started")
	ErrChallengeAlreadySubmitted = errors.New("challenge is already submitted")
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

	state, err := loadServiceState(datadir)
	if err != nil {
		if !errors.Is(err, ErrFileIsMissing) {
			return nil, err
		}
		state = newServiceState()
		if err := state.save(datadir); err != nil {
			return nil, fmt.Errorf("failed to save state: %w", err)
		}
	}

	cmds := make(chan Command, 1)

	privateKey := ed25519.NewKeyFromSeed(state.PrivKey[:32])
	s := &Service{
		commands: cmds,
		ServiceClient: ServiceClient{
			command: cmds,
		},
		cfg:             cfg,
		minMemoryLayer:  uint(minMemoryLayer),
		genesis:         genesis,
		datadir:         datadir,
		executingRounds: make(map[string]struct{}),
		privKey:         privateKey,
		PubKey:          privateKey.Public().(ed25519.PublicKey),
	}
	s.ServiceClient.serviceStarted = &s.started

	log.Info("Service public key: %x", s.PubKey)

	return s, nil
}

type roundResult struct {
	round *round
	err   error
}

func (s *Service) loop(ctx context.Context, roundsToResume []*round) {
	var eg errgroup.Group
	defer eg.Wait()

	logger := log.AppLog.WithName("worker")
	ctx = logging.NewContext(ctx, logger)

	roundResults := make(chan roundResult, 1)

	// Resume recovered rounds
	for _, round := range roundsToResume {
		round := round
		s.executingRounds[round.ID] = struct{}{}
		end := s.roundEndTime(round)
		eg.Go(func() error {
			err := round.recoverExecution(ctx, round.stateCache.Execution, end)
			roundResults <- roundResult{round: round, err: err}
			return nil
		})
	}

	timer := s.scheduleRound(ctx, s.openRound)

	for {
		select {
		case cmd := <-s.commands:
			cmd(s)

		case result := <-roundResults:
			if result.err == nil {
				broadcaster := s.broadcaster
				go broadcastProof(s, result.round, result.round.execution, broadcaster)
			} else {
				logger.With().Warning("round execution failed", log.Err(result.err), log.String("round", result.round.ID))
			}
			delete(s.executingRounds, result.round.ID)

		case <-timer:
			round := s.openRound
			s.openRound = s.newRound(ctx, round.Epoch()+1)
			s.executingRounds[round.ID] = struct{}{}

			end := s.roundEndTime(round)
			minMemoryLayer := s.minMemoryLayer
			eg.Go(func() error {
				err := round.execute(ctx, end, minMemoryLayer)
				roundResults <- roundResult{round, err}
				return nil
			})

			// schedule the next round
			timer = s.scheduleRound(ctx, s.openRound)

		case <-ctx.Done():
			logger.Info("service shutting down")
			return
		}
	}
}

func (s *Service) roundStartTime(round *round) time.Time {
	return s.genesis.Add(s.cfg.PhaseShift).Add(s.cfg.EpochDuration * time.Duration(round.Epoch()))
}

func (s *Service) roundEndTime(round *round) time.Time {
	return s.roundStartTime(round).Add(s.cfg.EpochDuration).Add(-s.cfg.CycleGap)
}

func (s *Service) scheduleRound(ctx context.Context, round *round) <-chan time.Time {
	waitTime := time.Until(s.roundStartTime(round))
	timer := time.After(waitTime)
	if waitTime > 0 {
		logging.FromContext(ctx).With().Info("waiting for execution to start", log.Duration("wait time", waitTime), log.String("round", round.ID))
	}
	return timer
}

func (s *Service) Start(b Broadcaster, verifier challenge_verifier.Verifier) error {
	s.Lock()
	defer s.Unlock()
	if s.Started() {
		return ErrAlreadyStarted
	}

	// Create a context for the running Service.
	// This context will be canceled when Service is stopped.
	ctx, stop := context.WithCancel(context.Background())
	s.stop = stop

	s.broadcaster = b

	var toResume []*round
	if s.cfg.NoRecovery {
		log.Info("Recovery is disabled")
	} else {
		var err error
		s.openRound, toResume, err = s.recover(ctx)
		if err != nil {
			return fmt.Errorf("failed to recover: %v", err)
		}
	}

	now := time.Now()
	epoch := time.Duration(0)
	if d := now.Sub(s.genesis); d > 0 {
		epoch = d / s.cfg.EpochDuration
	}
	if s.openRound == nil {
		s.openRound = s.newRound(ctx, uint32(epoch))
	}

	s.ServiceClient.SetChallengeVerifier(verifier)

	s.runningGroup.Go(func() error {
		s.loop(ctx, toResume)
		return nil
	})
	s.started.Store(true)
	return nil
}

// Shutdown gracefully stops running Service and waits
// for all processing to stop.
func (s *Service) Shutdown() error {
	if !s.Started() {
		return ErrNotStarted
	}
	log.Info("service shutting down")
	s.stop()
	err := s.runningGroup.Wait()
	s.started.Store(false)
	log.Info("service shutdown complete")
	return err
}

func (s *Service) Started() bool {
	return s.started.Load()
}

func (s *Service) recover(ctx context.Context) (open *round, executing []*round, err error) {
	logger := log.AppLog.WithName("recovery")
	logger.With().Info("Recovering service state", log.String("datadir", s.datadir))
	entries, err := os.ReadDir(s.datadir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		logger.Info("recovering entry %s", entry.Name())
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}
		r := newRound(ctx, s.datadir, uint32(epoch))
		state, err := r.state()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid round state: %v", err)
		}

		if state.isExecuted() {
			logger.Info("found round %v in executed state. broadcasting...", r.ID)
			go broadcastProof(s, r, state.Execution, s.broadcaster)
			continue
		}

		if state.isOpen() {
			logger.Info("found round %v in open state.", r.ID)
			if err := r.open(); err != nil {
				return nil, nil, fmt.Errorf("failed to open round: %v", err)
			}

			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			open = r
			continue
		}

		logger.Info("found round %v in executing state.", r.ID)
		executing = append(executing, r)
	}

	return open, executing, nil
}

func (s *ServiceClient) SetBroadcaster(b Broadcaster) {
	// No need to wait for the Command to execute.
	s.command <- func(s *Service) {
		old := s.broadcaster
		s.broadcaster = b
		if old != nil {
			log.Info("Service broadcaster updated")
		}
	}
}

func (s *ServiceClient) SetChallengeVerifier(provider challenge_verifier.Verifier) {
	s.challengeVerifier.Store(provider)
}

func (s *ServiceClient) Submit(ctx context.Context, challenge, signature []byte) (string, []byte, error) {
	if !s.serviceStarted.Load() {
		return "", nil, ErrNotStarted
	}
	logger := logging.FromContext(ctx)

	logger.Debug("Received challenge")
	// SAFETY: it will never panic as `s.ChallengeVerifier` is set in Start
	verifier := s.challengeVerifier.Load().(challenge_verifier.Verifier)
	result, err := verifier.Verify(ctx, challenge, signature)
	if err != nil {
		logger.With().Debug("challenge verification failed", log.Err(err))
		return "", nil, err
	}
	logger.With().Debug("verified challenge", log.String("hash", hex.EncodeToString(result.Hash)), log.String("node_id", hex.EncodeToString(result.NodeId)))

	type response struct {
		round string
		err   error
	}
	done := make(chan response, 1)
	s.command <- func(s *Service) {
		done <- response{
			round: s.openRound.ID,
			err:   s.openRound.submit(result.NodeId, result.Hash),
		}
		close(done)
	}

	select {
	case resp := <-done:
		switch {
		case errors.Is(resp.err, ErrChallengeAlreadySubmitted):
			return resp.round, result.Hash, nil
		case err != nil:
			return "", nil, resp.err
		}
		logger.With().Debug("submitted challenge for round", log.String("round", resp.round))
		return resp.round, result.Hash, nil
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
}

func (s *ServiceClient) Info(ctx context.Context) (*InfoResponse, error) {
	if !s.serviceStarted.Load() {
		return nil, ErrNotStarted
	}

	resp := make(chan *InfoResponse, 1)
	s.command <- func(s *Service) {
		defer close(resp)
		ids := make([]string, 0, len(s.executingRounds))
		for id := range s.executingRounds {
			ids = append(ids, id)
		}
		resp <- &InfoResponse{
			OpenRoundID:        s.openRound.ID,
			ExecutingRoundsIds: ids,
		}
	}
	select {
	case resp := <-resp:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// newRound creates a new round with the given epoch.
func (s *Service) newRound(ctx context.Context, epoch uint32) *round {
	r := newRound(ctx, s.datadir, epoch)
	if err := r.open(); err != nil {
		panic(fmt.Errorf("failed to open round: %v", err))
	}

	log.With().Info("Round opened", log.String("ID", r.ID))
	return r
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

// CreateChallengeVerifier creates a verifier connected to provided gateways.
// The verifier caches verification results and round-robins between gateways if
// verification fails.
func CreateChallengeVerifier(gateways []*grpc.ClientConn) (challenge_verifier.Verifier, error) {
	if len(gateways) == 0 {
		return nil, fmt.Errorf("need at least one gateway")
	}
	clients := make([]challenge_verifier.Verifier, 0, len(gateways))
	for _, target := range gateways {
		client := challenge_verifier.NewClient(target)
		clients = append(clients, client)
	}
	return challenge_verifier.NewCaching(
		ChallengeVerifierCacheSize, challenge_verifier.NewRetrying(
			challenge_verifier.NewRoundRobin(clients), 5, time.Second, 2))
}
