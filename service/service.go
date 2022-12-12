package service

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

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
	Genesis           string        `long:"genesis-time" description:"Genesis timestamp"`
	EpochDuration     time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift        time.Duration `long:"phase-shift"`
	CycleGap          time.Duration `long:"cycle-gap"`
	MemoryLayers      uint          `long:"memory" description:"Number of top Merkle tree layers to cache in-memory"`
	NoRecovery        bool          `long:"norecovery" description:"whether to disable a potential recovery procedure"`
	Reset             bool          `long:"reset" description:"whether to reset the service state by deleting the datadir"`
	GatewayAddresses  []string      `long:"gateway" description:"list of Spacemesh gateway nodes RPC listeners (host:port) for broadcasting of proofs"`
	ConnAcksThreshold uint          `long:"conn-acks" description:"number of required successful connections to Spacemesh gateway nodes"`
}

// estimatedLeavesPerSecond is used to computed estimated height of the proving tree
// in the epoch, which is used for cache estimation.
const estimatedLeavesPerSecond = 1 << 17

const ChallengeVerifierCacheSize = 1024

type serviceState struct {
	PrivKey []byte
}

// ServiceClient is an interface for interacting with the Service actor.
// It is created when the Service is started.
type ServiceClient struct {
	start             chan struct{}
	command           chan<- Command
	challengeVerifier atomic.Value // holds challenge_verifier.Verifier
}

// Service orchestrates rounds functionality
// It is responsible for accepting challenges, generating a proof from their hash digest and persisting it.
//
// `Service` is single-use, meaning it can be started with `Service::Run`.
// It is stopped by canceling the context provided to `Service::Run`.
// It mustn't be restarted. A new instance of `Service` must be created.
type Service struct {
	proofs   chan shared.ProofMessage
	commands <-chan Command
	ServiceClient

	cfg            *Config
	datadir        string
	genesis        time.Time
	minMemoryLayer uint

	// start channel is closed to trigger proofs generation
	start chan struct{}

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	openRound       *round
	executingRounds map[string]struct{}

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
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
	ErrRoundNotFinished          = errors.New("round is not finished yet")
)

// NewService creates a new instance of Poet Service.
// It should be started with `Service::Run`.
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
	cmds := make(chan Command, 1)

	privateKey := ed25519.NewKeyFromSeed(state.PrivKey[:32])

	roundsDir := filepath.Join(datadir, "rounds")
	if _, err := os.Stat(roundsDir); os.IsNotExist(err) {
		if err := os.Mkdir(roundsDir, 0o700); err != nil {
			return nil, err
		}
	}

	start := make(chan struct{})
	s := &Service{
		proofs:   make(chan shared.ProofMessage, 1),
		commands: cmds,
		ServiceClient: ServiceClient{
			start:   start,
			command: cmds,
		},
		start:           start,
		cfg:             cfg,
		minMemoryLayer:  uint(minMemoryLayer),
		genesis:         genesis,
		datadir:         datadir,
		executingRounds: make(map[string]struct{}),
		privKey:         privateKey,
		PubKey:          privateKey.Public().(ed25519.PublicKey),
	}

	log.Info("Service public key: %x", s.PubKey)

	return s, nil
}

type roundResult struct {
	round *round
	err   error
}

func (s *Service) ProofsChan() <-chan shared.ProofMessage {
	return s.proofs
}

func (s *Service) loop(ctx context.Context, roundsToResume []*round) {
	logger := logging.FromContext(ctx).WithName("worker")
	ctx = logging.NewContext(ctx, logger)

	// Make sure there is an open round
	if s.openRound == nil {
		epoch := uint32(0)
		if d := time.Since(s.genesis); d > 0 {
			epoch = uint32(d / s.cfg.EpochDuration)
		}
		s.openRound = s.newRound(ctx, uint32(epoch))
	}

	var eg errgroup.Group
	defer eg.Wait()

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

	var timer <-chan time.Time

	for {
		select {
		case <-s.start:
			logger.Info("starting proofs generation")
			timer = s.scheduleRound(ctx, s.openRound)
			s.start = nil

		case cmd := <-s.commands:
			cmd(s)

		case result := <-roundResults:
			if result.err == nil {
				s.reportNewProof(result.round.ID, result.round.execution)
			} else {
				logger.With().Error("round execution failed", log.Err(result.err), log.String("round", result.round.ID))
			}
			delete(s.executingRounds, result.round.ID)
			result.round.Close()

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

// Run starts the Service's actor event loop.
// It stops when the `ctx` is canceled.
func (s *Service) Run(ctx context.Context) error {
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

	s.loop(ctx, toResume)
	return nil
}

// Start starts proofs generation.
func (s *ServiceClient) Start(verifier challenge_verifier.Verifier) {
	s.SetChallengeVerifier(verifier)
	close(s.start)
}

// Started returns whether the `Service` is generating proofs.
func (s *ServiceClient) Started() bool {
	select {
	case _, ok := <-s.start:
		return !ok
	default:
		return false
	}
}

func (s *Service) recover(ctx context.Context) (open *round, executing []*round, err error) {
	roundsDir := filepath.Join(s.datadir, "rounds")
	logger := log.AppLog.WithName("recovery")
	logger.With().Info("Recovering service state", log.String("datadir", s.datadir))
	entries, err := os.ReadDir(roundsDir)
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
		r := newRound(ctx, roundsDir, uint32(epoch))
		state, err := r.state()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid round state: %v", err)
		}

		if state.isExecuted() {
			s.reportNewProof(r.ID, state.Execution)
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

func (s *ServiceClient) SetChallengeVerifier(provider challenge_verifier.Verifier) {
	s.challengeVerifier.Store(provider)
}

type SubmitResult struct {
	Round    string
	Hash     []byte
	RoundEnd time.Duration
}

func (s *ServiceClient) Submit(ctx context.Context, challenge, signature []byte) (*SubmitResult, error) {
	if !s.Started() {
		return nil, ErrNotStarted
	}
	logger := logging.FromContext(ctx)

	logger.Debug("Received challenge")
	// SAFETY: it will never panic as `s.ChallengeVerifier` is set in Start
	verifier := s.challengeVerifier.Load().(challenge_verifier.Verifier)
	result, err := verifier.Verify(ctx, challenge, signature)
	if err != nil {
		logger.With().Debug("challenge verification failed", log.Err(err))
		return nil, err
	}
	logger.With().Debug("verified challenge",
		log.String("hash", hex.EncodeToString(result.Hash)),
		log.String("node_id", hex.EncodeToString(result.NodeId)))

	type response struct {
		round string
		err   error
		end   time.Time
	}
	done := make(chan response, 1)
	s.command <- func(s *Service) {
		done <- response{
			round: s.openRound.ID,
			err:   s.openRound.submit(result.NodeId, result.Hash),
			end:   s.roundEndTime(s.openRound),
		}
		close(done)
	}

	select {
	case resp := <-done:
		switch {
		case resp.err == nil:
			logger.With().Debug("submitted challenge for round", log.String("round", resp.round))
		case errors.Is(resp.err, ErrChallengeAlreadySubmitted):
		case resp.err != nil:
			return nil, err
		}
		return &SubmitResult{
			Round:    resp.round,
			Hash:     result.Hash,
			RoundEnd: time.Until(resp.end),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *ServiceClient) Info(ctx context.Context) (*InfoResponse, error) {
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
	if err := saveState(s.datadir, s.privKey); err != nil {
		panic(err)
	}
	roundsDir := filepath.Join(s.datadir, "rounds")
	r := newRound(ctx, roundsDir, epoch)
	if err := r.open(); err != nil {
		panic(fmt.Errorf("failed to open round: %v", err))
	}

	log.With().Info("Round opened", log.String("ID", r.ID))
	return r
}

func (s *Service) reportNewProof(round string, execution *executionState) {
	s.proofs <- shared.ProofMessage{
		Proof: shared.Proof{
			MerkleProof: *execution.NIP,
			Members:     execution.Members,
			NumLeaves:   execution.NumLeaves,
		},
		ServicePubKey: s.PubKey,
		RoundID:       round,
		Signature:     nil,
	}
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
