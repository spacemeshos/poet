package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/jessevdk/go-flags"
	"github.com/spacemeshos/smutil/log"
	_ "github.com/syndtr/goleveldb/leveldb/table"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spacemeshos/poet/release/proto/go/rpc/api"
)

// Harness fully encapsulates an active poet server process to provide a unified
// platform to programmatically drive a poet server instance, whether for
// creating rpc driven integration tests, or for any other usage.
type Harness struct {
	server *server
	conn   *grpc.ClientConn
	api.PoetClient
}

// NewHarness creates and initializes a new instance of Harness.
func NewHarness(ctx context.Context, cfg *ServerConfig) (*Harness, error) {
	server, err := newServer(cfg)
	if err != nil {
		return nil, err
	}

	if isListening(cfg.rpcListen) {
		if err := killProcess(cfg.rpcListen); err != nil {
			return nil, err
		}
	}

	// Spawn a new poet server process.
	if err := server.start(); err != nil {
		return nil, err
	}

	// Verify the client connectivity.
	// If failed, shutdown the server.
	conn, err := connectClient(ctx, cfg.rpcListen)
	if err != nil {
		_ = server.shutdown(true)
		return nil, err
	}

	h := &Harness{
		server:     server,
		conn:       conn,
		PoetClient: api.NewPoetClient(conn),
	}

	return h, nil
}

// TearDown stops the harness running instance.
// The created process is killed, and the temporary
// directories are removed.
func (h *Harness) TearDown(cleanup bool) error {
	if err := h.conn.Close(); err != nil {
		log.Warning("failed to close connection: %v", err)
	}

	if err := h.server.shutdown(cleanup); err != nil {
		return fmt.Errorf("failed to shut down: %w", err)
	}

	return nil
}

// StderrPipe returns an stderr reader for the server process.
func (h *Harness) StderrPipe() io.Reader {
	return h.server.stderr
}

// StdoutPipe returns an stdout reader for the server process.
func (h *Harness) StdoutPipe() io.Reader {
	return h.server.stdout
}

// ProcessErrors returns a channel used for reporting any fatal process errors.
func (h *Harness) ProcessErrors() <-chan error {
	return h.server.errChan
}

// RESTListen returns the configured interface/port/socket for REST connections.
func (h *Harness) RESTListen() string {
	return h.server.cfg.RESTListen
}

// connectClient attempts to establish a gRPC Client connection
// to the provided target.
func connectClient(ctx context.Context, target string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server at %s: %v", target, err)
	}

	return conn, nil
}

// baseDir is the directory path of the temp directory for all the harness files.
func baseDir() (string, error) {
	baseDir := filepath.Join(os.TempDir(), "poet")
	err := os.MkdirAll(baseDir, 0o755)
	return baseDir, err
}

func isListening(addr string) bool {
	conn, _ := net.DialTimeout("tcp", addr, 1*time.Second)
	if conn != nil {
		_ = conn.Close()
		return true
	}
	return false
}

func killProcess(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		args := fmt.Sprintf("(Get-NetTCPConnection -LocalPort %d).OwningProcess -Force", addr.Port)
		cmd = exec.Command("Stop-Process", "-Id", args)
	} else {
		args := fmt.Sprintf("lsof -i tcp:%d | grep LISTEN | awk '{print $2}' | xargs kill -9", addr.Port)
		cmd = exec.Command("bash", "-c", args)
	}

	var errb bytes.Buffer
	cmd.Stderr = &errb

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error during killing process: %s | %s", err, errb.String())
	}

	return nil
}
