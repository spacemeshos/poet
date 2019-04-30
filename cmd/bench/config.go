package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

const (
	defaultN   = 10
	defaultCPU = false
)

// config defines the configuration options for bench.
type config struct {
	N   uint `short:"n" description:"protocol n param (table size = 2^n)"`
	CPU bool `short:"c" description:"whether to enable CPU profiling"`
}

// loadConfig initializes and parses the config using command line options.
func loadConfig() (*config, error) {
	// Default config.
	cfg := config{
		N:   defaultN,
		CPU: defaultCPU,
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
