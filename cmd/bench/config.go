package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

const defaultDuration = 10 * time.Second

// config defines the configuration options for bench.
type config struct {
	Duration   time.Duration `short:"d" description:"benchmark duration"`
	CpuProfile *string       `short:"c" description:"path to CPU profile file (disabled if not set)"`
	Memprofile *string       `short:"m" description:"path to memory profile file (disabled if not set)"`
}

// loadConfig initializes and parses the config using command line options.
func loadConfig() (*config, error) {
	// Default config.
	cfg := config{
		Duration: defaultDuration,
	}

	// Parse command line options.
	if _, err := flags.Parse(&cfg); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		return nil, err
	}

	return &cfg, nil
}
