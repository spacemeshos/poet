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
	Genesis       Genesis       `long:"genesis-time"   description:"Genesis timestamp in RFC3339 format"`
	EpochDuration time.Duration `long:"epoch-duration" description:"Epoch duration"`
	PhaseShift    time.Duration `long:"phase-shift"`
	CycleGap      time.Duration `long:"cycle-gap"`

	InitialPowChallenge string `long:"pow-challenge"  description:"The initial PoW challenge for the first round"`
	PowDifficulty       uint   `long:"pow-difficulty" description:"PoW difficulty (in the number of leading zero bits)"`

	MaxRoundMembers uint `long:"max-round-members" description:"the maximum number of members in a round"`

	// Merkle-Tree related configuration:
	EstimatedLeavesPerSecond uint `long:"lps"              description:"Estimated number of leaves generated per second"`
	MemoryLayers             uint `long:"memory"           description:"Number of top Merkle tree layers to cache in-memory"`
	TreeFileBufferSize       uint `long:"tree-file-buffer" description:"The size of memory buffer for file-based tree layers"`
}

type Genesis time.Time

// UnmarshalFlag implements flags.Unmarshaler.
func (g *Genesis) UnmarshalFlag(value string) error {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return err
	}
	*g = Genesis(t)
	return nil
}

func (g Genesis) Time() time.Time {
	return time.Time(g)
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
	minMemoryLayer uint

	// openRound is the round which is currently open for accepting challenges registration from miners.
	// At any given time there is one single open round.
	openRound        *round
	executingRoundId *string

	powVerifiers powVerifiers

	PubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey
}

// Command is a function that will be run in the main Service loop.
// Commands are run serially hence they don't require additional synchronization.
// The functions cannot block and should be kept short to not block the Service loop.
type Command func(*Service)

type InfoResponse struct {
	OpenRoundID      string
	ExecutingRoundId *string
}

var (
	ErrNotStarted                = errors.New("service not started")
	ErrAlreadyStarted            = errors.New("already started")
	ErrChallengeAlreadySubmitted = errors.New("challenge is already submitted")
	ErrConflictingRegistration   = errors.New("conflicting registration")
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

	estimatedLeaves := uint64(cfg.EpochDuration.Seconds()-cfg.CycleGap.Seconds()) * uint64(cfg.EstimatedLeavesPerSecond)
	minMemoryLayer := uint(0)
	if totalLayers := mshared.RootHeightFromWidth(estimatedLeaves); totalLayers > cfg.MemoryLayers {
		minMemoryLayer = totalLayers - cfg.MemoryLayers
	}

	logger := logging.FromContext(ctx)
	logger.Sugar().Infof("creating poet service. min memory layer: %v. genesis: %s", minMemoryLayer, cfg.Genesis.Time())
	logger.Info(
		"epoch configuration",
		zap.Duration("duration", cfg.EpochDuration),
		zap.Duration("cycle gap", cfg.CycleGap),
		zap.Duration("phase shift", cfg.PhaseShift),
	)

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
		proofs:         make(chan proofMessage, 1),
		commands:       cmds,
		cfg:            cfg,
		minMemoryLayer: minMemoryLayer,
		datadir:        datadir,
		privKey:        privateKey,
		PubKey:         privateKey.Public().(ed25519.PublicKey),
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
		if d := time.Since(s.cfg.Genesis.Time().Add(s.cfg.PhaseShift)); d > 0 {
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
		s.executingRoundId = &round.ID
		end := s.roundEndTime(round)
		eg.Go(func() error {
			unlock := lockOSThread(ctx, roundTidFile)
			defer unlock()
			err := round.recoverExecution(ctx, end, s.cfg.TreeFileBufferSize, 0)
			roundResults <- roundResult{round: round, err: err}
			return nil
		})
	}

	for {
		select {
		case result := <-roundResults:
			switch {
			case result.err == nil:
				s.onNewProof(result.round.ID, result.round.execution)
			case errors.Is(result.err, context.Canceled):
				logger.Info("round canceled", zap.String("round", result.round.ID))
			default:
				logger.Error("round failed", zap.Error(result.err), zap.String("round", result.round.ID))
			}

			eg.Go(func() error {
				if err := result.round.teardown(ctx, result.err == nil); err != nil {
					logger.Warn("round teardown failed", zap.Error(err), zap.String("round", result.round.ID))
				}
				return nil
			})
			s.executingRoundId = nil

		case cmd := <-s.commands:
			cmd(s)

		case <-s.timer:
			round := s.openRound
			newRound, err := s.newRound(ctx, round.Epoch()+1)
			if err != nil {
				return fmt.Errorf("failed to open new round: %w", err)
			}
			s.openRound = newRound
			s.executingRoundId = &round.ID

			end := s.roundEndTime(round)
			minMemoryLayer := s.minMemoryLayer
			eg.Go(func() error {
				unlock := lockOSThread(ctx, roundTidFile)
				defer unlock()
				err := round.execute(ctx, end, minMemoryLayer, s.cfg.TreeFileBufferSize, 0)
				roundResults <- roundResult{round, err}
				return nil
			})

			// schedule the next round
			s.timer = s.scheduleRound(ctx, s.openRound)

		case <-ctx.Done():
			logger.Info("service shutting down")
			_ = eg.Wait()
			// Process all finished rounds before exiting and finally teardown the open round.
			for {
				select {
				default:
					if err := s.openRound.teardown(ctx, false); err != nil {
						return fmt.Errorf("tearing down open round: %w", err)
					}
					return nil
				case result := <-roundResults:
					switch {
					case result.err == nil:
						s.onNewProof(result.round.ID, result.round.execution)
					case errors.Is(result.err, context.Canceled):
						logger.Info("round canceled", zap.String("round", result.round.ID))
					default:
						logger.Error("round failed", zap.Error(result.err), zap.String("round", result.round.ID))
					}
					if err := result.round.teardown(ctx, result.err == nil); err != nil {
						logger.Warn("round teardown failed", zap.Error(err), zap.String("round", result.round.ID))
					}
				}
			}
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
	return s.cfg.Genesis.Time().Add(s.cfg.PhaseShift).Add(s.cfg.EpochDuration * time.Duration(round.Epoch()))
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

	var err error
	s.openRound, toResume, err = s.recover(ctx)
	if err != nil {
		return fmt.Errorf("failed to recover: %v", err)
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

func (s *Service) recover(ctx context.Context) (open, executing *round, err error) {
	roundsDir := filepath.Join(s.datadir, "rounds")
	logger := logging.FromContext(ctx).Named("recovery")
	logger.Info("recovering service state", zap.String("datadir", s.datadir))
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
		r, err := newRound(roundsDir, uint32(epoch), s.cfg.MaxRoundMembers)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create round: %w", err)
		}

		err = r.loadState()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid round state: %w", err)
		}

		logger.Info("recovered round", zap.String("ID", r.ID), zap.Uint("members", r.members))

		switch {
		case r.isExecuted():
			logger.Info(
				"round is finished already",
				zap.String("ID", r.ID),
				zap.Time("started", r.executionStarted),
				zap.Binary("root", r.execution.NIP.Root),
				zap.Uint64("num_leaves", r.execution.NumLeaves),
			)
			s.onNewProof(r.ID, r.execution)
		case r.isOpen():
			logger.Info("round is open", zap.String("ID", r.ID))
			// Keep the last open round as openRound (multiple open rounds state is possible
			// only if recovery was previously disabled).
			if open != nil {
				logger.Warn("found more than 1 open round - overwriting", zap.String("previous", open.ID))
			}
			open = r
		default:
			// Round is in executing state.
			logger.Info(
				"round is executing",
				zap.String("ID", r.ID),
				zap.Time("started", r.executionStarted),
				zap.Uint64("num_leaves", r.execution.NumLeaves),
			)
			if executing != nil {
				logger.Warn("found more than 1 executing round - overwriting", zap.String("previous", executing.ID))
			}
			executing = r
		}
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
			err:   s.openRound.submit(ctx, nodeID, challenge),
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
			return nil, resp.err
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
		resp <- &InfoResponse{
			OpenRoundID:      s.openRound.ID,
			ExecutingRoundId: s.executingRoundId,
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
	r, err := newRound(roundsDir, epoch, s.cfg.MaxRoundMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new round: %w", err)
	}

	if err := r.saveState(); err != nil {
		_ = r.teardown(ctx, true)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	logging.FromContext(ctx).Info("round opened", zap.String("ID", r.ID))
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
