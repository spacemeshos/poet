package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// serverConfig contains all the args and data required to launch a poet server
// instance  and connect to it via rpc client.
type serverConfig struct {
	logLevel             string
	rpcListen            string
	baseDir              string
	dataDir              string
	exe                  string
	NodeAddress          string
	InitialRoundDuration string
}

// DefaultConfig returns a newConfig with all default values.
func DefaultConfig() (*serverConfig, error) {
	baseDir, err := baseDir()
	if err != nil {
		return nil, err
	}

	poetPath, err := poetExecutablePath(baseDir)
	if err != nil {
		return nil, err
	}

	cfg := &serverConfig{
		logLevel:  "debug",
		rpcListen: "127.0.0.1:18550",
		baseDir:   baseDir,
		dataDir:   filepath.Join(baseDir, "datadir"),
		exe:       poetPath,
	}

	return cfg, nil
}

// genArgs generates a slice of command line arguments from serverConfig instance.
func (cfg *serverConfig) genArgs() []string {
	var args []string

	args = append(args, fmt.Sprintf("--datadir=%v", cfg.dataDir))
	args = append(args, fmt.Sprintf("--rpclisten=%v", cfg.rpcListen))
	if cfg.NodeAddress != "" {
		args = append(args, fmt.Sprintf("--nodeaddr=%v", cfg.NodeAddress))
	}
	if cfg.InitialRoundDuration != "" {
		args = append(args, fmt.Sprintf("--initialduration=%v", cfg.InitialRoundDuration))
	}

	return args
}

// server houses the necessary state required to configure, launch,
// and manage poet server process.
type server struct {
	cfg *serverConfig
	cmd *exec.Cmd

	// processExit is a channel that's closed once it's detected that the
	// process this instance is bound to has exited.
	processExit chan struct{}

	quit chan struct{}
	wg   sync.WaitGroup

	errChan chan error
}

// newNode creates a new poet server instance according to the passed cfg.
func newServer(cfg *serverConfig) (*server, error) {
	return &server{
		cfg:     cfg,
		errChan: make(chan error),
	}, nil
}

// start launches a new running process of poet server.
func (s *server) start() error {
	s.quit = make(chan struct{})

	args := s.cfg.genArgs()
	s.cmd = exec.Command(s.cfg.exe, args...)

	// Redirect stderr output to buffer
	var errb bytes.Buffer
	s.cmd.Stderr = &errb

	if err := s.cmd.Start(); err != nil {
		return err
	}

	// Launch a new goroutine which that bubbles up any potential fatal
	// process errors to errChan.
	s.processExit = make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		err := s.cmd.Wait()

		if err != nil {
			// Don't propagate 'signal: killed' error,
			// since it's an expected behavior.
			if !strings.Contains(err.Error(), "signal: killed") {
				s.errChan <- fmt.Errorf("%v\n%v\n", err, errb.String())
			}
		}

		// Signal any onlookers that this process has exited.
		close(s.processExit)
	}()

	return nil
}

// shutdown terminates the running poet server process, and cleans up
// all files/directories created by it.
func (s *server) shutdown() error {
	if err := s.stop(); err != nil {
		return err
	}

	if err := s.cleanup(); err != nil {
		return err
	}

	return nil
}

// stop kills the server running process, since it doesn't support
// RPC-driven stop functionality.
func (s *server) stop() error {
	// Do nothing if the process is not running.
	if s.processExit == nil {
		return nil
	}

	if err := s.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}

	close(s.quit)
	s.wg.Wait()

	s.quit = nil
	s.processExit = nil
	return nil
}

// cleanup cleans up the temporary files/directories created by the server process.
func (s *server) cleanup() error {
	return os.RemoveAll(s.cfg.dataDir)
}
