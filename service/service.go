package service

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	mshared "github.com/spacemeshos/merkle-tree/shared"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/service/tid"
)

type Config struct {
	Genesis       string        `long:"genesis-time"   description:"Genesis timestamp"`
	EpochDuration time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift    time.Duration `long:"phase-shift"`
	CycleGap      time.Duration `long:"cycle-gap"`
	NoRecovery    bool          `long:"norecovery"     description:"whether to disable a potential recovery procedure"`
	Reset         bool          `long:"reset"          description:"whether to reset the service state by deleting the datadir"`

	InitialPowChallenge string `long:"pow-challenge"  description:"The initial PoW challenge for the first round"`
	PowDifficulty       uint   `long:"pow-difficulty" description:"PoW difficulty (in the number of leading zero bits)"`

	// Merkle-Tree related configuration:
	EstimatedLeavesPerSecond uint `long:"lps"              description:"Estimated number of leaves generated per second"`
	MemoryLayers             uint `long:"memory"           description:"Number of top Merkle tree layers to cache in-memory"`
	TreeFileBufferSize       uint `long:"tree-file-buffer" description:"The size of memory buffer for file-based tree layers"`
}

// Service orchestrates rounds functionality
// It is responsible for accepting challenges, generating a proof from their hash digest and persisting it.
//
// `Service` is single-use, meaning it can be started with `Service::Run`.
// It is stopped by canceling the context provided to `Service::Run`.
// It mustn't be restarted. A new instance of `Service` must be created.
type Service struct {
	started  atomic.Bool
	proofs   chan proofMessage
	commands chan Command
	timer    <-chan time.Time

	cfg            *Config
	datadir        string
	genesis        time.Time
	minMemoryLayer uint

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	openRound       *round
	executingRounds map[string]struct{}

	powVerifiers powVerifiers

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

var (
	ErrNotStarted                = errors.New("service not started")
	ErrAlreadyStarted            = errors.New("already started")
	ErrChallengeAlreadySubmitted = errors.New("challenge is already submitted")
	ErrRoundNotFinished          = errors.New("round is not finished yet")
)

type option struct {
	verifier PowVerifier
}

type OptionFunc func(*option) error

func WithPowVerifier(v PowVerifier) OptionFunc {
	return func(o *option) error {
		if v == nil {
			return errors.New("pow verifier cannot be nil")
		}
		o.verifier = v
		return nil
	}
}

// NewService creates a new instance of Poet Service.
// It should be started with `Service::Run`.
func NewService(ctx context.Context, cfg *Config, datadir string, opts ...OptionFunc) (*Service, error) {
	options := option{
		verifier: NewPowVerifier(NewPowParams([]byte(cfg.InitialPowChallenge), cfg.PowDifficulty)),
	}
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return nil, err
		}
	}

	genesis, err := time.Parse(time.RFC3339, cfg.Genesis)
	if err != nil {
		return nil, err
	}
	estimatedLeaves := uint64(cfg.EpochDuration.Seconds()-cfg.CycleGap.Seconds()) * uint64(cfg.EstimatedLeavesPerSecond)
	minMemoryLayer := uint(0)
	if totalLayers := mshared.RootHeightFromWidth(estimatedLeaves); totalLayers > cfg.MemoryLayers {
		minMemoryLayer = totalLayers - cfg.MemoryLayers
	}

	logger := logging.FromContext(ctx)
	logger.Sugar().Infof("creating poet service. min memory layer: %v. genesis: %s", minMemoryLayer, cfg.Genesis)
	logger.Info(
		"epoch configuration",
		zap.Duration("duration", cfg.EpochDuration),
		zap.Duration("cycle gap", cfg.CycleGap),
		zap.Duration("phase shift", cfg.PhaseShift),
	)

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

	roundsDir := filepath.Join(datadir, "rounds")
	if _, err := os.Stat(roundsDir); os.IsNotExist(err) {
		if err := os.Mkdir(roundsDir, 0o700); err != nil {
			return nil, err
		}
	}

	s := &Service{
		proofs:          make(chan proofMessage, 1),
		commands:        cmds,
		cfg:             cfg,
		minMemoryLayer:  minMemoryLayer,
		genesis:         genesis,
		datadir:         datadir,
		executingRounds: make(map[string]struct{}),
		privKey:         privateKey,
		PubKey:          privateKey.Public().(ed25519.PublicKey),
		powVerifiers: powVerifiers{
			previous: nil,
			current:  options.verifier,
		},
	}

	logging.FromContext(ctx).Sugar().Infof("service public key: %x", s.PubKey)

	return s, nil
}

type roundResult struct {
	round *round
	err   error
}

func (s *Service) ProofsChan() <-chan proofMessage {
	return s.proofs
}

func (s *Service) loop(ctx context.Context, roundToResume *round) error {
	logger := logging.FromContext(ctx).Named("worker")
	ctx = logging.NewContext(ctx, logger)

	// Make sure there is an open round
	if s.openRound == nil {
		epoch := uint32(0)
		if d := time.Since(s.genesis.Add(s.cfg.PhaseShift)); d > 0 {
			epoch = uint32(d/s.cfg.EpochDuration) + 1
		}
		newRound, err := s.newRound(ctx, uint32(epoch))
		if err != nil {
			return fmt.Errorf("opening a round: %w", err)
		}
		s.openRound = newRound
	}

	var eg errgroup.Group
	defer eg.Wait()

	roundResults := make(chan roundResult, 1)

	// file that round's thread ID is written to
	roundTidFile := path.Join(s.datadir, "round.tid")

	// Resume recovered round if any
	if round := roundToResume; round != nil {
		s.executingRounds[round.ID] = struct{}{}
		end := s.roundEndTime(round)
		eg.Go(func() error {
			unlock := lockOSThread(ctx, roundTidFile)
			defer unlock()
			err := round.recoverExecution(ctx, end, s.cfg.TreeFileBufferSize)
			if err := round.teardown(err == nil); err != nil {
				logger.Warn("round teardown failed", zap.Error(err))
			}
			roundResults <- roundResult{round: round, err: err}
			return nil
		})
	}

	for {
		select {
		case cmd := <-s.commands:
			cmd(s)

		case result := <-roundResults:
			if result.err == nil {
				s.onNewProof(result.round.ID, result.round.execution)
			} else {
				logger.Error("round execution failed", zap.Error(result.err), zap.String("round", result.round.ID))
			}
			delete(s.executingRounds, result.round.ID)

		case <-s.timer:
			round := s.openRound
			newRound, err := s.newRound(ctx, round.Epoch()+1)
			if err != nil {
				return fmt.Errorf("failed to open new round: %w", err)
			}
			s.openRound = newRound
			s.executingRounds[round.ID] = struct{}{}

			end := s.roundEndTime(round)
			minMemoryLayer := s.minMemoryLayer
			eg.Go(func() error {
				unlock := lockOSThread(ctx, roundTidFile)
				defer unlock()
				err := round.execute(ctx, end, minMemoryLayer, s.cfg.TreeFileBufferSize)
				if err := round.teardown(err == nil); err != nil {
					logger.Warn("round teardown failed", zap.Error(err))
				}
				roundResults <- roundResult{round, err}
				return nil
			})

			// schedule the next round
			s.timer = s.scheduleRound(ctx, s.openRound)

		case <-ctx.Done():
			logger.Info("service shutting down")
			if err := s.openRound.teardown(false); err != nil {
				return fmt.Errorf("tearing down open round: %w", err)
			}
			return nil
		}
	}
}

// lockOSThread:
// - locks current goroutine to OS thread,
// - writes the current TID to `tidFile`.
// The caller must call the returned `unlock` to unlock OS thread.
func lockOSThread(ctx context.Context, tidFile string) (unlock func()) {
	runtime.LockOSThread()

	if err := os.WriteFile(tidFile, []byte(strconv.Itoa(tid.Gettid())), os.ModePerm); err != nil {
		logging.FromContext(ctx).Warn("failed to write goroutine thread id to file", zap.Error(err))
	}

	return runtime.UnlockOSThread
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
		logging.FromContext(ctx).
			Info("waiting for execution to start", zap.Duration("wait time", waitTime), zap.String("round", round.ID))
	}
	return timer
}

// Run starts the Service's actor event loop.
// It stops when the `ctx` is canceled.
func (s *Service) Run(ctx context.Context) error {
	var toResume *round
	if s.cfg.NoRecovery {
		logging.FromContext(ctx).Info("Recovery is disabled")
	} else {
		var err error
		s.openRound, toResume, err = s.recover(ctx)
		if err != nil {
			return fmt.Errorf("failed to recover: %v", err)
		}
	}

	return s.loop(ctx, toResume)
}

// Start starts proofs generation.
func (s *Service) Start(ctx context.Context) error {
	resp := make(chan error)
	s.commands <- func(s *Service) {
		defer close(resp)
		if s.Started() {
			resp <- ErrAlreadyStarted
		}
		s.timer = s.scheduleRound(ctx, s.openRound)
		s.started.Store(true)
	}
	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Started returns whether the `Service` is generating proofs.
func (s *Service) Started() bool {
	return s.started.Load()
}

func (s *Service) recover(ctx context.Context) (open *round, executing *round, err error) {
	roundsDir := filepath.Join(s.datadir, "rounds")
	logger := logging.FromContext(ctx).Named("recovery")
	logger.Info("Recovering service state", zap.String("datadir", s.datadir))
	entries, err := os.ReadDir(roundsDir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		logger.Sugar().Infof("recovering entry %s", entry.Name())
		if !entry.IsDir() {
			continue
		}

		epoch, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("entry is not a uint32 %s", entry.Name())
		}
		r, err := newRound(roundsDir, uint32(epoch))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create round: %w", err)
		}

		err = r.loadState()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid round state: %w", err)
		}

		if r.isExecuted() {
			s.onNewProof(r.ID, r.execution)
			continue
		}

		if r.isOpen() {
			logger.Info("found round in open state.", zap.String("ID", r.ID))
			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			open = r
			continue
		}

		logger.Info("found round in executing state.", zap.String("ID", r.ID))
		if executing != nil {
			logger.Warn("found more than 1 executing round - overwriting", zap.String("previous", executing.ID))
		}
		executing = r
	}

	return open, executing, nil
}

type SubmitResult struct {
	Round    string
	RoundEnd time.Duration
}

func (s *Service) PowParams() PowParams {
	return s.powVerifiers.Params()
}

func (s *Service) Submit(
	ctx context.Context,
	challenge, nodeID []byte,
	nonce uint64,
	powParams PowParams,
) (*SubmitResult, error) {
	if !s.Started() {
		return nil, ErrNotStarted
	}
	logger := logging.FromContext(ctx)

	err := s.powVerifiers.VerifyWithParams(challenge, nodeID, nonce, powParams)
	if err != nil {
		logger.Debug("challenge verification failed", zap.Error(err))
		return nil, err
	}
	logger.Debug("verified challenge", zap.String("node_id", hex.EncodeToString(nodeID)))

	type response struct {
		round string
		err   error
		end   time.Time
	}
	done := make(chan response, 1)
	s.commands <- func(s *Service) {
		done <- response{
			round: s.openRound.ID,
			err:   s.openRound.submit(nodeID, challenge),
			end:   s.roundEndTime(s.openRound),
		}
		close(done)
	}

	select {
	case resp := <-done:
		switch {
		case resp.err == nil:
			logger.Debug("submitted challenge for round", zap.String("round", resp.round))
		case errors.Is(resp.err, ErrChallengeAlreadySubmitted):
		case resp.err != nil:
			return nil, err
		}
		return &SubmitResult{
			Round:    resp.round,
			RoundEnd: time.Until(resp.end),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) Info(ctx context.Context) (*InfoResponse, error) {
	resp := make(chan *InfoResponse, 1)
	s.commands <- func(s *Service) {
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
func (s *Service) newRound(ctx context.Context, epoch uint32) (*round, error) {
	roundsDir := filepath.Join(s.datadir, "rounds")
	r, err := newRound(roundsDir, epoch)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new round: %w", err)
	}

	if err := r.saveState(); err != nil {
		_ = r.teardown(true)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	logging.FromContext(ctx).Info("Round opened", zap.String("ID", r.ID))
	return r, nil
}

func (s *Service) onNewProof(round string, execution *executionState) {
	// Rotate Proof of Work challenge.
	params := s.powVerifiers.Params()
	params.Challenge = execution.NIP.Root
	s.powVerifiers.SetParams(params)

	// Report
	s.proofs <- proofMessage{
		Proof: proof{
			MerkleProof: *execution.NIP,
			Members:     execution.Members,
			NumLeaves:   execution.NumLeaves,
		},
		ServicePubKey: s.PubKey,
		RoundID:       round,
	}
}
