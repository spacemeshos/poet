package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/smutil/log"
)

var (
	cfg *config
)

// poetMain is the true entry point for poet. This function is required since
// defers created in the top-level scope of a main method aren't executed if
// os.Exit() is called.
func poetMain() error {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Load configuration and parse command line. This function also
	// initializes logging and configures it accordingly.
	loadedConfig, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = loadedConfig

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

	if err := startServer(); err != nil {
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
