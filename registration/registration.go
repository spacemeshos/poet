package registration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/config/round_config"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
)

//go:generate mockgen -package mocks -destination mocks/registration.go . WorkerService

type WorkerService interface {
	ExecuteRound(ctx context.Context, epoch uint, membershipRoot []byte) error
	RegisterForProofs(ctx context.Context) <-chan shared.NIP
}

// Registration orchestrates rounds functionality
// It is responsible for:
//   - registering challenges,
//   - scheduling rounds,
//   - feeding workers with membership tree roots for proof generation,
//   - serving proofs.
type Registration struct {
	genesis  time.Time
	cfg      Config
	roundCfg round_config.Config
	dbdir    string
	privKey  ed25519.PrivateKey

	openRoundMutex sync.RWMutex
	openRound      *round

	db       *database
	roundDbs map[uint]*leveldb.DB

	powVerifiers powVerifiers
	workerSvc    WorkerService
}

type newRegistrationOptionFunc func(*newRegistrationOptions)

type newRegistrationOptions struct {
	powVerifier PowVerifier
	privKey     ed25519.PrivateKey
	roundCfg    round_config.Config
	cfg         Config
}

func WithPowVerifier(verifier PowVerifier) newRegistrationOptionFunc {
	return func(opts *newRegistrationOptions) {
		opts.powVerifier = verifier
	}
}

func WithPrivateKey(privKey ed25519.PrivateKey) newRegistrationOptionFunc {
	return func(opts *newRegistrationOptions) {
		opts.privKey = privKey
	}
}

func WithRoundConfig(roundCfg round_config.Config) newRegistrationOptionFunc {
	return func(opts *newRegistrationOptions) {
		opts.roundCfg = roundCfg
	}
}

func WithConfig(cfg Config) newRegistrationOptionFunc {
	return func(opts *newRegistrationOptions) {
		opts.cfg = cfg
	}
}

func NewRegistration(
	ctx context.Context,
	genesis time.Time,
	dbdir string,
	workerSvc WorkerService,
	opts ...newRegistrationOptionFunc,
) (*Registration, error) {
	options := newRegistrationOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if options.privKey == nil {
		logging.FromContext(ctx).Info("generating new keys")
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("generating private key: %w", err)
		}
		options.privKey = priv
	}

	dbPath := filepath.Join(dbdir, "proofs")
	db, err := newDatabase(dbPath, options.privKey.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("opening proofs database: %w", err)
	}

	r := &Registration{
		genesis:   genesis,
		cfg:       options.cfg,
		roundCfg:  options.roundCfg,
		dbdir:     dbdir,
		privKey:   options.privKey,
		db:        db,
		workerSvc: workerSvc,
		roundDbs:  make(map[uint]*leveldb.DB),
	}

	if options.powVerifier == nil {
		r.setupPowProviders(ctx)
	} else {
		r.powVerifiers = powVerifiers{current: options.powVerifier}
	}

	round, err := r.createRound(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating new round: %w", err)
	}
	r.openRound = round
	logging.FromContext(ctx).Info(
		"Opened round",
		zap.Uint("epoch", r.openRound.epoch),
		zap.Int("members", r.openRound.members),
	)

	return r, nil
}

func (r *Registration) Pubkey() ed25519.PublicKey {
	return r.privKey.Public().(ed25519.PublicKey)
}

func (r *Registration) Close() (res error) {
	for epoch, db := range r.roundDbs {
		if err := db.Close(); err != nil {
			res = errors.Join(err, fmt.Errorf("closing round db for epoch %d: %w", epoch, err))
		}
	}

	return errors.Join(res, r.db.Close(), r.openRound.Close())
}

func (r *Registration) closeRound(ctx context.Context) error {
	r.openRoundMutex.Lock()
	defer r.openRoundMutex.Unlock()
	root, err := r.openRound.calcMembershipRoot()
	if err != nil {
		return fmt.Errorf("calculating membership root: %w", err)
	}
	logging.FromContext(ctx).
		Info("closing round", zap.Uint("epoch", r.openRound.epoch), zap.Binary("root", root), zap.Int("members", r.openRound.members))
	if err := r.workerSvc.ExecuteRound(ctx, r.openRound.epoch, root); err != nil {
		return fmt.Errorf("closing round for epoch %d: %w", r.openRound.epoch, err)
	}
	round, err := r.createRound(ctx)
	if err != nil {
		return fmt.Errorf("creating new round: %w", err)
	}
	if err := r.openRound.Close(); err != nil {
		return fmt.Errorf("closing round: %w", err)
	}
	r.openRound = round
	return nil
}

func (r *Registration) setupPowProviders(ctx context.Context) {
	current, previous, err := r.db.GetPowChallenges(ctx)
	if err != nil {
		challenge := make([]byte, 32)
		rand.Read(challenge)
		r.powVerifiers = powVerifiers{current: NewPowVerifier(PowParams{challenge, r.cfg.PowDifficulty})}
		if err := r.db.SavePowChallenge(ctx, challenge); err != nil {
			logging.FromContext(ctx).Warn("failed to persist PoW challenge", zap.Error(err))
		}
	} else {
		r.powVerifiers = powVerifiers{
			current:  NewPowVerifier(PowParams{current, r.cfg.PowDifficulty}),
			previous: NewPowVerifier(PowParams{previous, r.cfg.PowDifficulty}),
		}
	}
}

func (r *Registration) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx).Named("registration")
	ctx = logging.NewContext(ctx, logger)

	proofs := r.workerSvc.RegisterForProofs(ctx)

	if err := r.recoverExecution(ctx); err != nil {
		return fmt.Errorf("recovering execution: %w", err)
	}

	timer := r.scheduleRound(ctx, r.openRound.epoch)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer:
			if err := r.closeRound(ctx); err != nil {
				logger.Error("failed to close round", zap.Error(err))
			}
			timer = r.scheduleRound(ctx, r.openRound.epoch)
		case nip := <-proofs:
			if err := r.onNewProof(ctx, nip); err != nil {
				logger.Error("failed to process new proof", zap.Error(err), zap.Uint("epoch", nip.Epoch))
			}
		}
	}
}

func (r *Registration) recoverExecution(ctx context.Context) error {
	// First re-execute the in-progress round if any
	_, executing := r.Info(ctx)
	if executing == nil {
		return nil
	}
	db, err := leveldb.OpenFile(r.roundDbPath(*executing), &opt.Options{ErrorIfMissing: true})
	switch {
	case os.IsNotExist(err):
		// nothing to do
		return nil
	case err != nil:
		return err
	}

	round := newRound(*executing, db, r.newRoundOpts()...)
	logging.FromContext(ctx).Info("found round in progress, scheduling it", zap.Uint("epoch", round.epoch))
	root, err := round.calcMembershipRoot()
	if err != nil {
		return fmt.Errorf("calculating membership root: %w", err)
	}
	if err := r.workerSvc.ExecuteRound(ctx, round.epoch, root); err != nil {
		return fmt.Errorf("scheduling in-progress round for epoch %d: %w", round.epoch, err)
	}
	return nil
}

func (r *Registration) onNewProof(ctx context.Context, proof shared.NIP) error {
	logger := logging.FromContext(ctx).Named("on-proof").With(zap.Uint("epoch", proof.Epoch))
	logger.Info("received new proof", zap.Uint64("leaves", proof.Leaves))

	var (
		db *leveldb.DB
		ok bool
	)
	if db, ok = r.roundDbs[proof.Epoch]; !ok {
		// try to reopen round db
		var err error
		db, err = leveldb.OpenFile(r.roundDbPath(proof.Epoch), &opt.Options{ErrorIfMissing: true})
		if err != nil {
			return fmt.Errorf("reopening round db: %w", err)
		}
	}

	round := newRound(proof.Epoch, db, r.newRoundOpts()...)

	members := round.getMembers()
	if err := r.db.SaveProof(ctx, proof, members); err != nil {
		return fmt.Errorf("saving proof in DB: %w", err)
	} else {
		logger.Info("proof saved in DB", zap.Int("members", len(members)), zap.Uint64("leaves", proof.Leaves))
	}

	// Rotate Proof of Work challenge.
	params := r.powVerifiers.Params()
	params.Challenge = proof.Root
	r.powVerifiers.SetParams(params)
	membersMetric.DeleteLabelValues(epochToRoundId(proof.Epoch))
	return nil
}

func (r *Registration) newRoundOpts() []newRoundOptionFunc {
	return []newRoundOptionFunc{
		withMaxMembers(r.cfg.MaxRoundMembers),
		withMaxSubmitBatchSize(r.cfg.MaxSubmitBatchSize),
		withSubmitFlushInterval(r.cfg.SubmitFlushInterval),
	}
}

func (r *Registration) roundDbPath(epoch uint) string {
	return filepath.Join(r.dbdir, "rounds", epochToRoundId(epoch))
}

func (r *Registration) createRound(ctx context.Context) (*round, error) {
	epoch := r.roundCfg.OpenRoundId(r.genesis, time.Now())
	var db *leveldb.DB
	var ok bool
	if db, ok = r.roundDbs[epoch]; !ok {
		var err error
		db, err = leveldb.OpenFile(r.roundDbPath(epoch), nil)
		if err != nil {
			return nil, fmt.Errorf("opening round db: %w", err)
		}
		r.roundDbs[epoch] = db
	}
	round := newRound(epoch, db, r.newRoundOpts()...)

	logging.FromContext(ctx).
		Info("created new round", zap.Uint("epoch", epoch), zap.Time("start", r.roundCfg.RoundStart(r.genesis, epoch)))
	return round, nil
}

func (r *Registration) Submit(
	ctx context.Context,
	challenge, nodeID []byte,
	nonce uint64,
	powParams PowParams,
) (epoch uint, roundEnd time.Time, err error) {
	logger := logging.FromContext(ctx)

	err = r.powVerifiers.VerifyWithParams(challenge, nodeID, nonce, powParams)
	if err != nil {
		logger.Debug("challenge verification failed", zap.Error(err))
		return 0, time.Time{}, err
	}
	logger.Debug("verified challenge", zap.String("node_id", hex.EncodeToString(nodeID)))

	r.openRoundMutex.RLock()
	defer r.openRoundMutex.RUnlock()
	epoch = r.openRound.epoch
	done, err := r.openRound.submit(ctx, nodeID, challenge)
	switch {
	case err == nil:
		logger.Debug("async-submitted challenge for round", zap.Uint("round", epoch))
		// wait for actually submitted
		select {
		case <-ctx.Done():
			logger.Debug("context canceled while waiting for challenge to be submitted")
			return 0, time.Time{}, ctx.Err()
		case err := <-done:
			if err != nil {
				logger.Warn("challenge registration failed", zap.Error(err))
				return 0, time.Time{}, err
			}
		}
	case errors.Is(err, ErrChallengeAlreadySubmitted):
	case err != nil:
		return 0, time.Time{}, err
	}

	return epoch, r.genesis.Add(r.roundCfg.PhaseShift).Add(time.Duration(epoch+1) * r.roundCfg.EpochDuration), nil
}

func (r *Registration) Info(ctx context.Context) (openRoundId uint, executingRoundId *uint) {
	r.openRoundMutex.RLock()
	defer r.openRoundMutex.RUnlock()
	openRoundId = r.openRound.epoch
	if openRoundId > 0 {
		executingRoundId = new(uint)
		*executingRoundId = uint(int(openRoundId) - 1)
	}
	return openRoundId, executingRoundId
}

func (r *Registration) PowParams() PowParams {
	return r.powVerifiers.Params()
}

func (r *Registration) Proof(ctx context.Context, roundId string) (*proofData, error) {
	return r.db.GetProof(ctx, roundId)
}

func epochToRoundId(epoch uint) string {
	return strconv.FormatUint(uint64(epoch), 10)
}

func (r *Registration) scheduleRound(ctx context.Context, epoch uint) <-chan time.Time {
	waitTime := time.Until(r.roundCfg.RoundStart(r.genesis, epoch))
	timer := time.After(waitTime)
	if waitTime > 0 {
		logging.FromContext(ctx).
			Info("waiting for execution to start", zap.Duration("wait time", waitTime), zap.Uint("epoch", epoch))
	}
	return timer
}
