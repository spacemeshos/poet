// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2017-2019 The Spacemesh developers

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"

	"github.com/spacemeshos/poet/appdata"
	"github.com/spacemeshos/poet/service"
)

const (
	defaultConfigFilename           = "poet.conf"
	defaultDataDirname              = "data"
	defaultLogDirname               = "logs"
	defaultLogFilename              = "poet.log"
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
	defaultPoetDir       = appdata.AppDataDir("poet", false)
	defaultConfigFile    = filepath.Join(defaultPoetDir, defaultConfigFilename)
	defaultDataDir       = filepath.Join(defaultPoetDir, defaultDataDirname)
	defaultLogDir        = filepath.Join(defaultPoetDir, defaultLogDirname)
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
	return &Config{
		PoetDir:         defaultPoetDir,
		ConfigFile:      defaultConfigFile,
		DataDir:         defaultDataDir,
		LogDir:          defaultLogDir,
		MaxLogFiles:     defaultMaxLogFiles,
		MaxLogFileSize:  defaultMaxLogFileSize,
		RawRPCListener:  fmt.Sprintf("localhost:%d", defaultRPCPort),
		RawRESTListener: fmt.Sprintf("localhost:%d", defaultRESTPort),
		Service: &service.Config{
			Genesis:                  time.Now().Format(time.RFC3339),
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

// ReadConfigFile reads values from a conf file.
func ReadConfigFile(cfg *Config) (*Config, error) {
	cfg.PoetDir = cleanAndExpandPath(cfg.PoetDir)
	cfg.ConfigFile = cleanAndExpandPath(cfg.ConfigFile)
	if cfg.PoetDir != defaultPoetDir {
		if cfg.ConfigFile == defaultConfigFile {
			cfg.ConfigFile = filepath.Join(
				cfg.PoetDir, defaultConfigFilename,
			)
		}
	}

	if err := flags.IniParse(cfg.ConfigFile, cfg); err != nil {
		return nil, fmt.Errorf("failed to read config from file: %w", err)
	}

	return cfg, nil
}

// SetupConfig initializes filesystem and network infrastructure.
func SetupConfig(cfg *Config) (*Config, error) {
	// If the provided poet directory is not the default, we'll modify the
	// path to all of the files and directories that will live within it.
	if cfg.PoetDir != defaultPoetDir {
		cfg.DataDir = filepath.Join(cfg.PoetDir, defaultDataDirname)
		cfg.LogDir = filepath.Join(cfg.PoetDir, defaultLogDirname)
	}

	// Create the poet directory if it doesn't already exist.
	if err := os.MkdirAll(cfg.PoetDir, 0o700); err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		var pathError *fs.PathError
		if errors.As(err, &pathError) && errors.Is(err, fs.ErrExist) {
			if link, lerr := os.Readlink(pathError.Path); lerr == nil {
				err = fmt.Errorf("is symlink %s -> %s mounted?", pathError.Path, link)
			}
		}
		return nil, fmt.Errorf("failed to create poet directory: %w", err)
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
