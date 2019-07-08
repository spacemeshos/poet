package integration

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spacemeshos/poet/rpc/api"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Harness fully encapsulates an active poet server process to provide a unified
// platform to programmatically drive a poet server instance, whether for
// creating rpc driven integration tests, or for any other usage.
type Harness struct {
	server *server
	api.PoetClient
}

// NewHarness creates and initializes a new instance of Harness.
// nodeAddress (string) is the address of a Spacemesh node gRPC server. Use "NO_BROADCAST" to skip broadcasting proofs.
func NewHarness(cfg *serverConfig) (*Harness, error) {
	server, err := newServer(cfg)
	if err != nil {
		return nil, err
	}

	if isListening(cfg.rpcListen) {
		err = killProcess(cfg.rpcListen)
		if err != nil {
			return nil, err
		}
	}

	// Spawn a new poet server process.
	if err := server.start(); err != nil {
		return nil, err
	}

	// Verify the client connectivity.
	// If failed, shutdown the server.
	conn, err := connectClient(cfg.rpcListen)
	if err != nil {
		_ = server.shutdown()
		return nil, err
	}

	h := &Harness{
		server:     server,
		PoetClient: api.NewPoetClient(conn),
	}

	return h, nil
}

// TearDown stops the harness running instance.
// The created process is killed, and the temporary
// directories are removed.
func (h *Harness) TearDown() error {
	if err := h.server.shutdown(); err != nil {
		return err
	}

	return nil
}

// ProcessErrors returns a channel used for reporting any fatal process errors.
func (h *Harness) ProcessErrors() <-chan error {
	return h.server.errChan
}

// connectClient attempts to establish a gRPC Client connection
// to the provided target.
func connectClient(target string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server at %s: %v", target, err)
	}

	return conn, nil
}

// baseDir is the directory path of the temp directory for all the harness files.
func baseDir() (string, error) {
	baseDir := filepath.Join(os.TempDir(), "poet")
	err := os.MkdirAll(baseDir, 0755)
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
		args := fmt.Sprintf("(Get-NetTCPConnection -LocalPort %s).OwningProcess -Force", addr.Port)
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
