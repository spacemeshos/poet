package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"

	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/smutil/log"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/server"
)

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

	log.JSONLog(cfg.JSONLog)
	log.InitSpacemeshLoggingSystem(cfg.LogDir, "poet.log")

	defer func() {
		log.Info("Shutdown complete")
	}()

	// Show version at startup.
	log.Info("Version: %s, dir: %v, datadir: %v, genesis: %v", version(), cfg.PoetDir, cfg.DataDir, cfg.Service.Genesis)

	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		log.Info("Starting HTTP profiling on port %v", cfg.Profile)
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
			log.Error("Could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Error("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := server.StartServer(ctx, cfg); err != nil {
		log.Error("failed to start server: %v", err)
		return err
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
