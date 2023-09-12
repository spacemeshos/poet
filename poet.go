package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" //#nosec G108 -- DefaultServeMux is not used
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/migrations"
	"github.com/spacemeshos/poet/server"
)

// Poet binary version.
// It should be passed during the build with '-ldflags "-X main.version="'.
var version = "unknown"

// poetMain is the true entry point for poet. This function is required since
// defers created in the top-level scope of a main method aren't executed if
// os.Exit() is called.
func poetMain() (err error) {
	// Start with a default Config with sane settings
	cfg := server.DefaultConfig()
	// Pre-parse the command line to check for an alternative Config file
	cfg, err = server.ParseFlags(cfg)
	if err != nil {
		return err
	}
	// Load configuration file overwriting defaults with any specified options
	// Parse CLI options and overwrite/add any specified options
	cfg, err = server.ReadConfigFile(cfg)
	if err != nil {
		return err
	}

	cfg, err = server.SetupConfig(cfg)
	if err != nil {
		return err
	}
	// Finally, parse the remaining command line options again to ensure
	// they take precedence.
	cfg, err = server.ParseFlags(cfg)
	if err != nil {
		return err
	}

	// Initialize logging
	logLevel := zap.InfoLevel
	if cfg.DebugLog {
		logLevel = zap.DebugLevel
	}
	logger := logging.New(logLevel, filepath.Join(cfg.LogDir, "poet.log"), cfg.JSONLog)
	ctx := logging.NewContext(context.Background(), logger)

	defer func() {
		logger.Info("shutdown complete")
	}()

	// Show version at startup.
	logger.Sugar().
		Infof("version: %s, dir: %v, datadir: %v, genesis: %v", version, cfg.PoetDir, cfg.DataDir, cfg.Genesis.Time())

	// Migrate data if needed
	if err := migrations.Migrate(ctx, cfg); err != nil {
		return fmt.Errorf("migrations failed: %w", err)
	}

	if cfg.MetricsPort != nil {
		// Start Prometheus
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", *cfg.MetricsPort))
		if err != nil {
			return err
		}
		logger.Info("spawning Prometheus", zap.String("endpoint", fmt.Sprintf("http://%s", lis.Addr().String())))
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())
			server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second * 5}
			err := server.Serve(lis)
			logger.With().Info("Metrics server stopped", zap.Error(err))
		}()
	}
	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		logger.Sugar().Info("starting HTTP profiling on port %v", cfg.Profile)
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			server := &http.Server{Addr: listenAddr, ReadHeaderTimeout: time.Second * 5}
			if err := server.ListenAndServe(); err != nil {
				logger.Warn("Failed to start profiling HTTP server", zap.Error(err))
			}
		}()
	} else {
		// Disable go default unbounded memory profiler.
		runtime.MemProfileRate = 0
	}

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			logger.Error("could not create CPU profile", zap.Error(err))
		}
		defer func() {
			err = errors.Join(err, f.Close())
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Error("could not start CPU profile", zap.Error(err))
		}
		defer pprof.StopCPUProfile()
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	server, err := server.New(ctx, *cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := server.Start(ctx); err != nil {
		errClosing := server.Close()
		return fmt.Errorf("failure in server: %w (closing: %w)", err, errClosing)
	}

	if err := server.Close(); err != nil {
		return fmt.Errorf("failed closing server: %w", err)
	}
	return nil
}

func main() {
	// Call the "real" main in a nested manner so the defers will properly
	// be executed in the case of a graceful shutdown.
	if err := poetMain(); err != nil {
		// If it's the flag utility error don't print it,
		// because it was already printed.
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
