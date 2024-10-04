package registration_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	"github.com/spacemeshos/poet/registration/mocks"
	"github.com/spacemeshos/poet/server"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/transport"
)

func TestSubmitIdempotence(t *testing.T) {
	req := require.New(t)
	genesis := time.Now().Add(time.Second)

	roundCfg := server.RoundConfig{
		EpochDuration: time.Hour,
		PhaseShift:    time.Second / 2,
		CycleGap:      time.Second / 4,
	}

	challenge := []byte("challenge")
	nodeID := []byte("nodeID")
	nonce := uint64(7)

	verifier := mocks.NewMockPowVerifier(gomock.NewController(t))
	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&roundCfg,
		registration.WithPowVerifier(verifier),
	)
	req.NoError(err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	verifier.EXPECT().Params().Times(2).Return(registration.PowParams{})
	verifier.EXPECT().Verify(challenge, nodeID, nonce).Times(2).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	// Submit challenge
	epoch, _, err := r.Submit(
		context.Background(),
		challenge,
		nodeID,
		nonce,
		registration.PowParams{},
		nil,
		nil,
		time.Time{},
	)
	req.NoError(err)
	req.Equal(uint(0), epoch)

	// Try again - it should return the same result
	epoch, _, err = r.Submit(
		context.Background(), challenge,
		nodeID, nonce, registration.PowParams{}, nil, nil, time.Time{})
	req.NoError(err)
	req.Equal(uint(0), epoch)

	cancel()
	req.NoError(eg.Wait())
}

func TestOpeningRounds(t *testing.T) {
	t.Parallel()
	t.Run("before genesis", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 0.
		require.Equal(t, uint(0), reg.OpenRound())
	})
	t.Run("after genesis, but within phase shift", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{PhaseShift: time.Minute * 10},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 0.
		require.Equal(t, uint(0), reg.OpenRound())
	})
	t.Run("in first epoch", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(-time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute * 10,
			},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 1.
		require.Equal(t, uint(1), reg.OpenRound())
	})
	t.Run("in distant epoch", func(t *testing.T) {
		t.Parallel()
		reg, err := registration.New(
			context.Background(),
			time.Now().Add(-100*time.Hour),
			t.TempDir(),
			nil,
			&server.RoundConfig{
				EpochDuration: time.Hour,
				PhaseShift:    time.Minute,
			},
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reg.Close()) })

		// Service instance should create open round 100.
		require.Equal(t, uint(100), reg.OpenRound())
	})
}

func TestWorkingWithoutWorkerService(t *testing.T) {
	t.Parallel()

	reg, err := registration.New(
		context.Background(),
		time.Now(),
		t.TempDir(),
		transport.NewInMemory(),
		&server.RoundConfig{EpochDuration: time.Millisecond * 10},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reg.Close()) })

	var eg errgroup.Group
	ctx, cancel := context.WithCancel(logging.NewContext(context.Background(), zaptest.NewLogger(t)))
	defer cancel()
	eg.Go(func() error { return reg.Run(ctx) })

	// Verify that registration keeps opening rounds even without the worker service.
	// Only check if the round number is incremented to not rely on the exact timing.
	for i := 0; i < 3; i++ {
		round := reg.OpenRound()
		require.Eventually(t, func() bool { return reg.OpenRound() > round }, time.Second, time.Millisecond*10)
	}
	cancel()
	require.NoError(t, eg.Wait())
}

// Test if Proof of Work challenge is rotated every round.
// The challenge should be changed to the root of PoET proof Merkle tree
// of the previous round.
func TestPowChallengeRotation(t *testing.T) {
	genesis := time.Now()

	proofs := make(chan shared.NIP, 1)

	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(proofs)
	workerSvc.EXPECT().
		ExecuteRound(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, epoch uint, _ []byte) error {
			select {
			case proofs <- shared.NIP{
				MerkleProof: shared.MerkleProof{
					Root: []byte{1, 2, 3, 4},
				},
				Epoch: epoch,
			}:
			default:
			}
			return nil
		}).
		AnyTimes()

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&server.RoundConfig{EpochDuration: 10 * time.Millisecond},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	params0 := r.PowParams()
	require.NotEqual(t, []byte{1, 2, 3, 4}, params0.Challenge)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	require.Eventually(t, func() bool {
		return !bytes.Equal([]byte{1, 2, 3, 4}, r.PowParams().Challenge)
	}, time.Second, time.Millisecond)

	cancel()
	require.NoError(t, eg.Wait())
}

func TestRecoveringRoundInProgress(t *testing.T) {
	req := require.New(t)
	genesis := time.Now()

	roundCfg := server.RoundConfig{
		EpochDuration: time.Hour,
		PhaseShift:    time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	verifier := mocks.NewMockPowVerifier(gomock.NewController(t))
	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))
	workerSvc.EXPECT().ExecuteRound(gomock.Any(), gomock.Eq(uint(0)), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint, _ []byte) error {
			cancel()
			return nil
		},
	)

	r, err := registration.New(
		context.Background(),
		genesis,
		t.TempDir(),
		workerSvc,
		&roundCfg,
		registration.WithPowVerifier(verifier),
	)
	req.NoError(err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })

	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error { return r.Run(ctx) })

	req.NoError(eg.Wait())

	// Restart the registration service.
	// The round in progress should be recovered and executed again.
	ctx, cancel = context.WithCancel(context.Background())
	workerSvc.EXPECT().RegisterForProofs(gomock.Any()).Return(make(<-chan shared.NIP, 1))
	workerSvc.EXPECT().ExecuteRound(gomock.Any(), gomock.Eq(uint(0)), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint, _ []byte) error {
			cancel()
			return nil
		},
	)
	req.NoError(r.Run(ctx))
}

func makeCert(tb testing.TB, signingKey ed25519.PrivateKey, nodeID []byte, expiration *time.Time) *shared.OpaqueCert {
	data, err := shared.EncodeCert(&shared.Cert{Pubkey: nodeID, Expiration: expiration})
	require.NoError(tb, err)
	return &shared.OpaqueCert{
		Data:      data,
		Signature: ed25519.Sign(signingKey, data),
	}
}

func Test_GetCertifierInfo(t *testing.T) {
	certifier := &registration.CertifierConfig{
		PubKey: registration.Base64Enc("pubkey"),
		URL:    "http://the-certifier.org",
	}

	r, err := registration.New(
		context.Background(),
		time.Now(),
		t.TempDir(),
		nil,
		server.DefaultRoundConfig(),
		registration.WithConfig(registration.Config{
			MaxRoundMembers: 10,
			Certifier:       certifier,
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })
	require.Equal(t, r.CertifierInfo(), certifier)
}

func Test_CheckCertificate(t *testing.T) {
	challenge := []byte("challenge")
	nodeID := []byte("nodeID00nodeID00nodeID00nodeID00")

	submit := func(r *registration.Registration, cert *shared.OpaqueCert, hint *shared.CertKeyHint) error {
		_, _, err := r.Submit(
			context.Background(),
			challenge,
			nodeID,
			5,
			r.PowParams(),
			hint,
			cert,
			time.Time{},
		)
		return err
	}

	t.Run("certification check disabled (default config)", func(t *testing.T) {
		powVerifier := mocks.NewMockPowVerifier(gomock.NewController(t))
		powVerifier.EXPECT().Params().Return(registration.PowParams{}).AnyTimes()
		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
			registration.WithPowVerifier(powVerifier),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		// missing certificate - fallback to PoW
		powVerifier.EXPECT().Verify(challenge, nodeID, uint64(5)).Return(nil)
		require.NoError(t, submit(r, nil, nil))

		// passed certificate - still fallback to PoW
		_, private, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		cert := makeCert(t, private, nodeID, nil)

		powVerifier.EXPECT().Verify(challenge, nodeID, uint64(5)).Return(nil)
		require.NoError(t, submit(r, cert, nil))
	})
	t.Run("certification check enabled", func(t *testing.T) {
		certKeyPub, certKey, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		pubKeyHint := shared.CertKeyHint(certKeyPub)

		trustedKeysDir := t.TempDir()
		trustedKeys := generateTrustedKeys(t, 3, trustedKeysDir)

		trustedKeyPub, trustedKey := trustedKeys[0].Public().(ed25519.PublicKey), trustedKeys[0]
		trustedKeyHint := shared.CertKeyHint(trustedKeyPub)
		trustedKeyCert := makeCert(t, trustedKey, nodeID, nil)
		defaultValidCert := makeCert(t, certKey, nodeID, nil)

		powVerifier := mocks.NewMockPowVerifier(gomock.NewController(t))
		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
			registration.WithPowVerifier(powVerifier),
			registration.WithConfig(registration.Config{
				MaxRoundMembers: 10,
				Certifier: &registration.CertifierConfig{
					PubKey:             registration.Base64Enc(certKeyPub),
					TrustedKeysDirPath: trustedKeysDir,
				},
			}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		t.Run("missing certificate - fallback to PoW", func(t *testing.T) {
			powVerifier.EXPECT().Params().Return(registration.PowParams{}).AnyTimes()
			powVerifier.EXPECT().Verify(challenge, nodeID, uint64(5)).Return(nil)
			require.NoError(t, submit(r, nil, nil))
		})
		t.Run("valid certificate (unexpired)", func(t *testing.T) {
			require.NoError(t, submit(r, defaultValidCert, &pubKeyHint))
		})
		t.Run("valid certificate (expired)", func(t *testing.T) {
			expiration := time.Now().Add(-time.Hour)
			certificate := makeCert(t, certKey, nodeID, &expiration)
			require.ErrorIs(t, submit(r, certificate, &pubKeyHint), registration.ErrInvalidCertificate)
		})

		t.Run("valid certificate (no public key hint, use main pubkey as default)", func(t *testing.T) {
			require.NoError(t, submit(r, defaultValidCert, nil))
		})

		t.Run("valid certificate (invalid public key hint)", func(t *testing.T) {
			hint := &shared.CertKeyHint{1, 2, 3, 4}
			require.ErrorIs(t, submit(r, defaultValidCert, hint), registration.ErrInvalidCertificate)
		})

		t.Run("valid certificate signed by a trusted key", func(t *testing.T) {
			require.NoError(t, submit(r, trustedKeyCert, &trustedKeyHint))
		})

		t.Run("invalid certificate (wrong node ID)", func(t *testing.T) {
			_, _, err = r.Submit(
				context.Background(),
				challenge,
				[]byte("wrong node ID"),
				0,
				r.PowParams(),
				&pubKeyHint,
				defaultValidCert,
				time.Time{},
			)
			require.ErrorIs(t, err, registration.ErrInvalidCertificate)
		})
		t.Run("invalid certificate (signature)", func(t *testing.T) {
			_, wrongPrivate, err := ed25519.GenerateKey(nil)
			require.NoError(t, err)
			certificate := makeCert(t, wrongPrivate, nodeID, nil)
			require.ErrorIs(t, submit(r, certificate, &pubKeyHint), registration.ErrInvalidCertificate)
		})
	})
	t.Run("using only the main key", func(t *testing.T) {
		certKeyPub, certKeypriv, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)

		encodedCert, err := shared.EncodeCert(&shared.Cert{Pubkey: nodeID})
		require.NoError(t, err)
		cert := &shared.OpaqueCert{
			Data:      encodedCert,
			Signature: ed25519.Sign(certKeypriv, encodedCert),
		}

		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
			registration.WithConfig(registration.Config{
				MaxRoundMembers: 10,
				Certifier: &registration.CertifierConfig{
					PubKey: registration.Base64Enc(certKeyPub),
				},
			}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		t.Run("submit with a hint", func(t *testing.T) {
			hint := shared.CertKeyHint(certKeyPub)
			require.NoError(t, submit(r, cert, &hint))
		})
		t.Run("submit without a hint", func(t *testing.T) {
			require.NoError(t, submit(r, cert, nil))
		})
	})
	t.Run("using only trusted keys", func(t *testing.T) {
		trustedKeysDir := t.TempDir()
		trustedKeys := generateTrustedKeys(t, 3, trustedKeysDir)
		trustedKeyPub, trustedKey := trustedKeys[0].Public().(ed25519.PublicKey), trustedKeys[0]
		trustedKeyHint := shared.CertKeyHint(trustedKeyPub)

		validcert := makeCert(t, trustedKey, nodeID, nil)

		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
			registration.WithConfig(registration.Config{
				MaxRoundMembers: 10,
				Certifier: &registration.CertifierConfig{
					// Pubkey is not configured - it  will use only the keys loaded from the configured dirtory
					TrustedKeysDirPath: trustedKeysDir,
				},
			}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		t.Run("can't submit without a hint", func(t *testing.T) {
			require.ErrorIs(t, submit(r, validcert, nil), registration.ErrInvalidCertificate)
		})
		t.Run("submit with a hint finds a trusted key", func(t *testing.T) {
			require.NoError(t, submit(r, validcert, &trustedKeyHint))
		})
		t.Run("submit with invalid hint fails", func(t *testing.T) {
			require.ErrorIs(t, submit(r, validcert, &shared.CertKeyHint{}), registration.ErrNoMatchingCertPublicKeys)
		})
	})
}

func generateTrustedKeys(t *testing.T, n int, path string) (keys []ed25519.PrivateKey) {
	for i := range n {
		pubKey, priv, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		keys = append(keys, priv)

		path := filepath.Join(path, fmt.Sprintf("valid_key_%d.key", i))
		require.NoError(t, os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(pubKey)), 0o644))
	}
	return keys
}

func TestLoadTrustedKeys(t *testing.T) {
	genesis := time.Now()
	workerSvc := mocks.NewMockWorkerService(gomock.NewController(t))

	t.Run("certification check enabled", func(t *testing.T) {
		dir := t.TempDir()
		r, err := registration.New(
			context.Background(),
			genesis,
			t.TempDir(),
			workerSvc,
			server.DefaultRoundConfig(),
			registration.WithConfig(
				registration.Config{
					Certifier: &registration.CertifierConfig{
						TrustedKeysDirPath: dir,
					},
				}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		t.Run("load valid public keys", func(t *testing.T) {
			generateTrustedKeys(t, 3, dir)
			err = r.LoadTrustedPublicKeys(context.Background())
			require.NoError(t, err)
		})

		t.Run("load invalid public keys", func(t *testing.T) {
			path := filepath.Join(dir, "invalid_key.key")
			require.NoError(t, os.WriteFile(path, []byte("invalid"), 0o644))

			err = r.LoadTrustedPublicKeys(context.Background())
			require.ErrorIs(t, err, registration.ErrInvalidPublicKey)
		})
	})

	t.Run("certification check disabled", func(t *testing.T) {
		r, err := registration.New(
			context.Background(),
			time.Now(),
			t.TempDir(),
			nil,
			server.DefaultRoundConfig(),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		err = r.LoadTrustedPublicKeys(context.Background())
		require.ErrorIs(t, err, registration.ErrCertificationIsNotSupported)
	})

	t.Run("keys dir isn't configured", func(t *testing.T) {
		r, err := registration.New(
			context.Background(),
			genesis,
			t.TempDir(),
			workerSvc,
			server.DefaultRoundConfig(),
			registration.WithConfig(
				registration.Config{
					Certifier: &registration.CertifierConfig{},
				}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		err = r.LoadTrustedPublicKeys(context.Background())
		require.ErrorIs(t, registration.ErrTrustedKeyDirPathIsNotSet, err)
	})

	t.Run("Loading new trusted keys overwrites existing keys", func(t *testing.T) {
		dir := t.TempDir()
		keys := generateTrustedKeys(t, 1, dir)
		r, err := registration.New(
			context.Background(),
			genesis,
			t.TempDir(),
			workerSvc,
			server.DefaultRoundConfig(),
			registration.WithConfig(
				registration.Config{
					MaxRoundMembers: 10,
					Certifier: &registration.CertifierConfig{
						TrustedKeysDirPath: dir,
					},
				}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Close()) })

		trustedKey := keys[0]
		pubkeyHint := shared.CertKeyHint(trustedKey.Public().(ed25519.PublicKey))

		nodeID := []byte("nodeID")
		encodedCert, err := shared.EncodeCert(&shared.Cert{Pubkey: nodeID})
		require.NoError(t, err)
		validcert := &shared.OpaqueCert{
			Data:      encodedCert,
			Signature: ed25519.Sign(trustedKey, encodedCert),
		}

		// submit with a valid key
		_, _, err = r.Submit(
			context.Background(),
			[]byte("challenge"),
			nodeID,
			0,
			r.PowParams(),
			&pubkeyHint,
			validcert,
			time.Time{},
		)
		require.NoError(t, err)

		// create and load new keys
		generateTrustedKeys(t, 1, dir)
		err = r.LoadTrustedPublicKeys(context.Background())
		require.NoError(t, err)

		// submit the same cert again - it should fail because the certifier key isn't recognized anymore.
		_, _, err = r.Submit(
			context.Background(),
			[]byte("challenge"),
			nodeID,
			0,
			r.PowParams(),
			&pubkeyHint,
			validcert,
			time.Time{},
		)
		require.ErrorIs(t, err, registration.ErrNoMatchingCertPublicKeys)
	})
}
