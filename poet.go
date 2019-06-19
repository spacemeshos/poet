package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/go-spacemesh/log"
	"os"
	"runtime"
)

var (
	cfg *config
)

// poetMain is the true entry point for poet. This function is required since
// defers created in the top-level scope of a main method aren't executed if
// os.Exit() is called.
func poetMain() error {
	// Load configuration and parse command line. This function also
	// initializes logging and configures it accordingly.
	loadedConfig, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = loadedConfig
	defer func() {
		log.Info("Shutdown complete")
	}()

	// Show version at startup.
	log.Info("Version: %s", version())

	if err := startServer(); err != nil {
		return err
	}

	return nil
}

func main() {
	// Disable go default unbounded memory profiler
	runtime.MemProfileRate = 0

	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

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
