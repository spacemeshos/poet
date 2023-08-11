// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2017-2019 The Spacemesh developers

package config

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/service"
)

const (
	defaultDataDirname              = "data"
	defaultLogDirname               = "logs"
	defaultMaxLogFiles              = 3
	defaultMaxLogFileSize           = 10
	defaultRPCPort                  = 50002
	defaultRESTPort                 = 8080
	defaultMemoryLayers             = 26 // Up to (1 << 26) * 2 - 1 Merkle tree cache nodes (32 bytes each) will be held in-memory
	defaultTreeFileBufferSize       = 4096
	defaultEstimatedLeavesPerSecond = 78000
	defaultMaxRoundMembers          = 1 << 32
)

var (
	defaultEpochDuration = 30 * time.Second
	defaultPhaseShift    = 5 * time.Second
	defaultCycleGap      = 5 * time.Second
)

// Config defines the configuration options for poet.
//
// See loadConfig for further details regarding the
// configuration loading+parsing process.
type Config struct {
	PoetDir         string  `long:"poetdir"        description:"The base directory that contains poet's data, logs, configuration file, etc."`
	ConfigFile      string  `long:"configfile"     description:"Path to configuration file"                                                   short:"c"`
	DataDir         string  `long:"datadir"        description:"The directory to store poet's data within"                                    short:"b"`
	DbDir           string  `long:"dbdir"          description:"The directory to store DBs within. Defaults to datadir"`
	LogDir          string  `long:"logdir"         description:"Directory to log output."`
	DebugLog        bool    `long:"debuglog"       description:"Enable debug logs"`
	JSONLog         bool    `long:"jsonlog"        description:"Whether to log in JSON format"`
	MaxLogFiles     int     `long:"maxlogfiles"    description:"Maximum logfiles to keep (0 for no rotation)"`
	MaxLogFileSize  int     `long:"maxlogfilesize" description:"Maximum logfile size in MB"`
	RawRPCListener  string  `long:"rpclisten"      description:"The interface/port/socket to listen for RPC connections"                      short:"r"`
	RawRESTListener string  `long:"restlisten"     description:"The interface/port/socket to listen for REST connections"                     short:"w"`
	MetricsPort     *uint16 `long:"metrics-port"   description:"The port to expose metrics"`

	CPUProfile string `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	Profile    string `long:"profile"    description:"Enable HTTP profiling on given port -- must be between 1024 and 65535"`

	Service *service.Config `group:"Service"`
}

// DefaultConfig returns a config with default hardcoded values.
func DefaultConfig() *Config {
	poetDir := "./poet"
	cacheDir, err := os.UserCacheDir()
	if err == nil {
		poetDir = filepath.Join(cacheDir, "poet")
	}

	return &Config{
		PoetDir:         poetDir,
		DataDir:         filepath.Join(poetDir, defaultDataDirname),
		DbDir:           filepath.Join(poetDir, defaultDataDirname),
		LogDir:          filepath.Join(poetDir, defaultLogDirname),
		MaxLogFiles:     defaultMaxLogFiles,
		MaxLogFileSize:  defaultMaxLogFileSize,
		RawRPCListener:  fmt.Sprintf("localhost:%d", defaultRPCPort),
		RawRESTListener: fmt.Sprintf("localhost:%d", defaultRESTPort),
		Service: &service.Config{
			Genesis:                  service.Genesis(time.Now()),
			EpochDuration:            defaultEpochDuration,
			PhaseShift:               defaultPhaseShift,
			CycleGap:                 defaultCycleGap,
			MemoryLayers:             defaultMemoryLayers,
			TreeFileBufferSize:       defaultTreeFileBufferSize,
			EstimatedLeavesPerSecond: defaultEstimatedLeavesPerSecond,
			MaxRoundMembers:          defaultMaxRoundMembers,
		},
	}
}

// ParseFlags reads values from command line arguments.
func ParseFlags(preCfg *Config) (*Config, error) {
	if _, err := flags.Parse(preCfg); err != nil {
		return nil, err
	}
	return preCfg, nil
}

// ReadConfigFile reads config from an ini file.
// It uses the provided `cfg` as a base config and overrides it with the values
// from the config file.
func ReadConfigFile(cfg *Config) (*Config, error) {
	if cfg.ConfigFile == "" {
		return cfg, nil
	}
	logging.FromContext(context.Background()).Sugar().Debugf("reading config from %s", cfg.ConfigFile)
	if err := flags.IniParse(cfg.ConfigFile, cfg); err != nil {
		return nil, fmt.Errorf("failed to read config from %v: %w", cfg.ConfigFile, err)
	}

	return cfg, nil
}

// SetupConfig expands paths and initializes filesystem.
func SetupConfig(cfg *Config) (*Config, error) {
	// If the provided poet directory is not the default, we'll modify the
	// path to all of the files and directories that will live within it.
	defaultCfg := DefaultConfig()
	if cfg.PoetDir != defaultCfg.DataDir {
		if cfg.DataDir == defaultCfg.DataDir {
			cfg.DataDir = filepath.Join(cfg.PoetDir, defaultDataDirname)
		}
		if cfg.LogDir == defaultCfg.LogDir {
			cfg.LogDir = filepath.Join(cfg.PoetDir, defaultLogDirname)
		}
		if cfg.DbDir == defaultCfg.DbDir {
			cfg.DbDir = filepath.Join(cfg.PoetDir, defaultDataDirname)
		}
	}

	// Create the poet directory if it doesn't already exist.
	if err := os.MkdirAll(cfg.PoetDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create %v: %w", cfg.PoetDir, err)
	}

	// As soon as we're done parsing configuration options, ensure all paths
	// to directories and files are cleaned and expanded before attempting
	// to use them later on.
	cfg.DataDir = cleanAndExpandPath(cfg.DataDir)
	cfg.LogDir = cleanAndExpandPath(cfg.LogDir)

	return cfg, nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// This function is taken from https://github.com/btcsuite/btcd
func cleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		user, err := user.Current()
		if err == nil {
			homeDir = user.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}
