// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2017-2019 The Spacemesh developers

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/smutil/log"

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
	defaultConnAcksThreshold        = 1
	defaultBroadcastAcksThreshold   = 1
	defaultBroadcastNumRetries      = 100
	defaultBroadcastRetriesInterval = 5 * time.Minute
	defaultGatewayConnectionTimeout = 30 * time.Second
)

var (
	defaultPoetDir       = appdata.AppDataDir("poet", false)
	defaultConfigFile    = filepath.Join(defaultPoetDir, defaultConfigFilename)
	defaultDataDir       = filepath.Join(defaultPoetDir, defaultDataDirname)
	defaultLogDir        = filepath.Join(defaultPoetDir, defaultLogDirname)
	defaultGenesisTime   = time.Now().Format(time.RFC3339)
	defaultEpochDuration = 30 * time.Second
	defaultPhaseShift    = 5 * time.Second
	defaultCycleGap      = 5 * time.Second
)

type coreServiceConfig struct {
	MemoryLayers uint `long:"memory" description:"Number of top Merkle tree layers to cache in-memory"`
}

// Config defines the configuration options for poet.
//
// See loadConfig for further details regarding the
// configuration loading+parsing process.
type Config struct {
	PoetDir         string `long:"poetdir" description:"The base directory that contains poet's data, logs, configuration file, etc."`
	ConfigFile      string `short:"c" long:"configfile" description:"Path to configuration file"`
	DataDir         string `short:"b" long:"datadir" description:"The directory to store poet's data within"`
	LogDir          string `long:"logdir" description:"Directory to log output."`
	DebugLog        bool   `long:"debuglog" description:"Enable debug logs"`
	JSONLog         bool   `long:"jsonlog" description:"Whether to log in JSON format"`
	MaxLogFiles     int    `long:"maxlogfiles" description:"Maximum logfiles to keep (0 for no rotation)"`
	MaxLogFileSize  int    `long:"maxlogfilesize" description:"Maximum logfile size in MB"`
	RawRPCListener  string `short:"r" long:"rpclisten" description:"The interface/port/socket to listen for RPC connections"`
	RawRESTListener string `short:"w" long:"restlisten" description:"The interface/port/socket to listen for REST connections"`
	RPCListener     net.Addr
	RESTListener    net.Addr
	GtwConnTimeout  time.Duration `long:"gtw-connection-timeout" description:"Timeout for connecting to gateway"`

	CPUProfile string `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	Profile    string `long:"profile" description:"Enable HTTP profiling on given port -- must be between 1024 and 65535"`

	CoreServiceMode bool `long:"core" description:"Enable poet in core service mode"`

	CoreService *coreServiceConfig `group:"Core Service" namespace:"core"`
	Service     *service.Config    `group:"Service"`
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
		GtwConnTimeout:  defaultGatewayConnectionTimeout,
		Service: &service.Config{
			Genesis:                  defaultGenesisTime,
			EpochDuration:            defaultEpochDuration,
			PhaseShift:               defaultPhaseShift,
			CycleGap:                 defaultCycleGap,
			MemoryLayers:             defaultMemoryLayers,
			ConnAcksThreshold:        defaultConnAcksThreshold,
			BroadcastAcksThreshold:   defaultBroadcastAcksThreshold,
			BroadcastNumRetries:      defaultBroadcastNumRetries,
			BroadcastRetriesInterval: defaultBroadcastRetriesInterval,
		},
		CoreService: &coreServiceConfig{
			MemoryLayers: defaultMemoryLayers,
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
func ReadConfigFile(preCfg *Config) (*Config, error) {
	preCfg.PoetDir = cleanAndExpandPath(preCfg.PoetDir)
	preCfg.ConfigFile = cleanAndExpandPath(preCfg.ConfigFile)
	if preCfg.PoetDir != defaultPoetDir {
		if preCfg.ConfigFile == defaultConfigFile {
			preCfg.ConfigFile = filepath.Join(
				preCfg.PoetDir, defaultConfigFilename,
			)
		}
	}

	// Next, load any additional configuration options from the file.
	var configFileError error
	cfg := preCfg
	if err := flags.IniParse(preCfg.ConfigFile, cfg); err != nil {
		// If it's a parsing related error, then we'll return
		// immediately, otherwise we can proceed as possibly the Config
		// file doesn't exist which is OK.
		var iniError *flags.IniError
		if errors.As(err, &iniError) {
			return nil, err
		}

		configFileError = err
	}

	// Warn about missing Config file only after all other configuration is
	// done. This prevents the warning on help messages and invalid
	// options.
	if configFileError != nil {
		log.Warning("%v", configFileError)
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
		if errors.As(err, &pathError) && os.IsExist(err) {
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

	// Resolve the RPC listener
	addr, err := net.ResolveTCPAddr("tcp", cfg.RawRPCListener)
	if err != nil {
		return nil, err
	}
	cfg.RPCListener = addr

	// Resolve the REST listener
	addr, err = net.ResolveTCPAddr("tcp", cfg.RawRESTListener)
	if err != nil {
		return nil, err
	}
	cfg.RESTListener = addr

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
