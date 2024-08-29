package registration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/shared"
)

//go:generate mockgen -package mocks -destination mocks/registration.go . WorkerService

type WorkerService interface {
	ExecuteRound(ctx context.Context, epoch uint, membershipRoot []byte) error
	RegisterForProofs(ctx context.Context) <-chan shared.NIP
}

type roundConfig interface {
	OpenRoundId(genesis, now time.Time) uint
	RoundStart(genesis time.Time, epoch uint) time.Time
	RoundEnd(genesis time.Time, epoch uint) time.Time
}

var (
	ErrInvalidCertificate          = errors.New("invalid certificate")
	ErrTooLateToRegister           = errors.New("too late to register for the desired round")
	ErrCertificationIsNotSupported = errors.New("certificate is not supported")
	ErrTrustedKeyFilePathIsNotSet  = errors.New("trusted keys directory path is not set in the configuration")
	ErrInvalidPublicKey            = errors.New("invalid public key")

	registerWithCertMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "registration",
		Name:      "with_cert_total",
		Help:      "Number of registrations with a certificate",
	}, []string{"result"})

	registerWithPoWMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "poet",
		Subsystem: "registration",
		Name:      "with_pow_total",
		Help:      "Number of registrations with a PoW",
	}, []string{"result"})
)

// Registration orchestrates rounds functionality
// It is responsible for:
//   - registering challenges,
//   - scheduling rounds,
//   - feeding workers with membership tree roots for proof generation,
//   - serving proofs.
type Registration struct {
	genesis  time.Time
	cfg      Config
	roundCfg roundConfig
	dbdir    string
	privKey  ed25519.PrivateKey

	trustedKeysMtx    sync.RWMutex
	trustedPublicKeys [][]byte

	openRoundMutex sync.RWMutex
	openRound      *round

	db *database

	powVerifiers powVerifiers
	workerSvc    WorkerService
}

type newRegistrationOptionFunc func(*newRegistrationOptions)

type newRegistrationOptions struct {
	powVerifier PowVerifier
	privKey     ed25519.PrivateKey
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

func WithConfig(cfg Config) newRegistrationOptionFunc {
	return func(opts *newRegistrationOptions) {
		opts.cfg = cfg
	}
}

func New(
	ctx context.Context,
	genesis time.Time,
	dbdir string,
	workerSvc WorkerService,
	roundCfg roundConfig,
	opts ...newRegistrationOptionFunc,
) (*Registration, error) {
	options := newRegistrationOptions{
		cfg: DefaultConfig(),
	}
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
		roundCfg:  roundCfg,
		dbdir:     dbdir,
		privKey:   options.privKey,
		db:        db,
		workerSvc: workerSvc,
	}

	if options.powVerifier == nil {
		r.setupPowProviders(ctx)
	} else {
		r.powVerifiers = powVerifiers{current: options.powVerifier}
	}

	if r.cfg.Certifier != nil && r.cfg.Certifier.PubKey != nil {
		logging.FromContext(ctx).Info("configured certifier", zap.Inline(r.cfg.Certifier))
	} else {
		logging.FromContext(ctx).Info("disabled certificate checking")
		r.cfg.Certifier = nil
	}

	epoch := r.roundCfg.OpenRoundId(r.genesis, time.Now())
	round, err := newRound(epoch, r.dbdir, r.newRoundOpts()...)
	if err != nil {
		return nil, fmt.Errorf("creating new round: %w", err)
	}
	logging.FromContext(ctx).Info("opened round", zap.Uint("epoch", epoch), zap.Int("members", round.members))
	r.openRound = round

	return r, nil
}

func (r *Registration) CertifierInfo() *CertifierConfig {
	return r.cfg.Certifier
}

func (r *Registration) LoadTrustedPublicKeys() error {
	if r.cfg.Certifier == nil {
		return ErrCertificationIsNotSupported
	}

	r.trustedKeysMtx.Lock()
	defer r.trustedKeysMtx.Unlock()

	r.trustedPublicKeys = [][]byte{}

	dirPath := r.cfg.Certifier.TrustedKeysDirPath
	if dirPath == "" {
		return ErrTrustedKeyFilePathIsNotSet
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".key" {
			return nil
		}

		keyData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		decodedKey, err := base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			return err
		}

		if len(decodedKey) != ed25519.PublicKeySize {
			return ErrInvalidPublicKey
		}

		r.trustedPublicKeys = append(r.trustedPublicKeys, decodedKey)
		return nil
	})
	return err
}

func (r *Registration) Pubkey() ed25519.PublicKey {
	return r.privKey.Public().(ed25519.PublicKey)
}

func (r *Registration) Close() error {
	return errors.Join(r.db.Close(), r.openRound.Close())
}

func (r *Registration) closeRound(ctx context.Context) error {
	r.openRoundMutex.Lock()
	defer r.openRoundMutex.Unlock()
	root, err := r.openRound.calcMembershipRoot()
	if err != nil {
		return fmt.Errorf("calculating membership root: %w", err)
	}
	logging.FromContext(ctx).Info("closing round",
		zap.Uint("epoch", r.openRound.epoch),
		zap.Binary("root", root),
		zap.Int("members", r.openRound.members),
	)

	if err := r.openRound.Close(); err != nil {
		logging.FromContext(ctx).Error("failed to close the open round", zap.Error(err))
	}
	if err := r.workerSvc.ExecuteRound(ctx, r.openRound.epoch, root); err != nil {
		return fmt.Errorf("closing round for epoch %d: %w", r.openRound.epoch, err)
	}
	epoch := r.roundCfg.OpenRoundId(r.genesis, time.Now())
	round, err := newRound(epoch, r.dbdir, r.newRoundOpts()...)
	if err != nil {
		return fmt.Errorf("creating new round: %w", err)
	}
	logging.FromContext(ctx).Info("opened round", zap.Uint("epoch", epoch), zap.Int("members", round.members))

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

	// First re-execute the in-progress round if any
	if r.openRound.epoch > 0 {
		if err := r.recoverExecution(ctx, r.openRound.epoch-1); err != nil {
			return fmt.Errorf("recovering execution: %w", err)
		}
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

func (r *Registration) recoverExecution(ctx context.Context, epoch uint) error {
	opts := append(r.newRoundOpts(), failIfNotExists())
	round, err := newRound(epoch, r.dbdir, opts...)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return err
	}
	defer round.Close()

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

	// Retrieve the list of round members for the round.
	// This is temporary until we remove the list of members from the proof.
	opts := append(r.newRoundOpts(), failIfNotExists())
	round, err := newRound(proof.Epoch, r.dbdir, opts...)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return err
	}

	members := round.getMembers()
	if err := round.Close(); err != nil {
		logger.Error("failed to close round", zap.Error(err))
	}
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

func (r *Registration) verifyCert(
	certificate *shared.OpaqueCert,
	certPubKeyHint, nodeID []byte,
) error {
	r.trustedKeysMtx.RLock()
	defer r.trustedKeysMtx.RUnlock()

	_, err := shared.VerifyCertificate(certificate, append(r.trustedPublicKeys, r.cfg.Certifier.PubKey.Bytes()), nodeID, certPubKeyHint)
	return err
}

func (r *Registration) Submit(
	ctx context.Context,
	challenge, certPubKeyHint, nodeID []byte,
	// TODO: remove deprecated PoW
	nonce uint64,
	powParams PowParams,
	certificate *shared.OpaqueCert,
	deadline time.Time,
) (epoch uint, roundEnd time.Time, err error) {
	logger := logging.FromContext(ctx)
	// Verify if the node is allowed to register.
	// Support both a certificate and a PoW while
	// the certificate path is being stabilized.
	if r.cfg.Certifier != nil && certificate != nil {
		err = r.verifyCert(certificate, certPubKeyHint, nodeID)
		switch {
		case errors.Is(err, shared.ErrCertExpired):
			registerWithCertMetric.WithLabelValues("expired").Inc()
			return 0, time.Time{}, errors.Join(ErrInvalidCertificate, err)
		case err != nil:
			registerWithCertMetric.WithLabelValues("invalid").Inc()
			return 0, time.Time{}, errors.Join(ErrInvalidCertificate, err)
		default:
			registerWithCertMetric.WithLabelValues("valid").Inc()
		}
	} else {
		// FIXME: PoW is deprecated
		// Remove once certificate path is stabilized and mandatory.
		err := r.powVerifiers.VerifyWithParams(challenge, nodeID, nonce, powParams)
		if err != nil {
			registerWithPoWMetric.WithLabelValues("invalid").Inc()
			logger.Debug("PoW verification failed", zap.Error(err))
			return 0, time.Time{}, err
		}
		registerWithPoWMetric.WithLabelValues("valid").Inc()
		logger.Debug("verified PoW", zap.String("node_id", hex.EncodeToString(nodeID)))
	}

	r.openRoundMutex.RLock()
	epoch = r.openRound.epoch
	endTime := r.roundCfg.RoundEnd(r.genesis, epoch)
	if !deadline.IsZero() && endTime.After(deadline) {
		r.openRoundMutex.RUnlock()
		logger.Debug(
			"rejecting registration as too late",
			zap.Uint("round", epoch),
			zap.Time("deadline", deadline),
			zap.Time("end_time", endTime),
		)
		return epoch, endTime, ErrTooLateToRegister
	}
	done, err := r.openRound.submit(ctx, nodeID, challenge)
	r.openRoundMutex.RUnlock()

	switch {
	case err == nil:
		logger.Debug("async-submitted challenge for round", zap.Uint("round", epoch))
		// wait for actually submitted
		select {
		case <-ctx.Done():
			return 0, time.Time{}, ctx.Err()
		case err := <-done:
			if err != nil {
				return 0, time.Time{}, err
			}
		}
	case errors.Is(err, ErrChallengeAlreadySubmitted):
	default: // err != nil
		return 0, time.Time{}, err
	}

	return epoch, endTime, nil
}

func (r *Registration) OpenRound() uint {
	r.openRoundMutex.RLock()
	defer r.openRoundMutex.RUnlock()
	return r.openRound.epoch
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
