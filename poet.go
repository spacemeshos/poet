package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/server"
)

// Poet binary version.
// It should be passed during the build with '-ldflags "-X main.version="'.
var version = "unknown"

// poetMain is the true entry point for poet. This function is required since
// defers created in the top-level scope of a main method aren't executed if
// os.Exit() is called.
func poetMain() error {
	var err error
	// Start with a default Config with sane settings
	cfg := config.DefaultConfig()
	// Pre-parse the command line to check for an alternative Config file
	cfg, err = config.ParseFlags(cfg)
	if err != nil {
		return err
	}
	// Load configuration file overwriting defaults with any specified options
	// Parse CLI options and overwrite/add any specified options
	cfg, err = config.ReadConfigFile(cfg)
	if err != nil {
		return err
	}

	cfg, err = config.SetupConfig(cfg)
	if err != nil {
		return err
	}
	// Finally, parse the remaining command line options again to ensure
	// they take precedence.
	cfg, err = config.ParseFlags(cfg)
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
	logger.Sugar().Infof("version: %s, dir: %v, datadir: %v, genesis: %v", version, cfg.PoetDir, cfg.DataDir, cfg.Service.Genesis)

	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		logger.Sugar().Info("starting HTTP profiling on port %v", cfg.Profile)
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			fmt.Println(http.ListenAndServe(listenAddr, nil))
		}()
	} else {
		// Disable go default unbounded memory profiler.
		runtime.MemProfileRate = 0
	}

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			logger.With(zap.Error(err)).Error("could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			logger.With(zap.Error(err)).Error("could not start CPU profile")
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
		return fmt.Errorf("failure in server: %w", err)
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
