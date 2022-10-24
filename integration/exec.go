package integration

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	// compileMtx guards access to the executable path so that the project is
	// only compiled once.
	compileMtx sync.Mutex

	// executablePath is the path to the compiled executable. This is an empty
	// string until the initial compilation. It should not be accessed directly;
	// use the poetExecutablePath() function instead.
	executablePath string
)

// poetExecutablePath returns a path to the poet server executable.
// To ensure the code tests against the most up-to-date version, this method
// compiles poet server the first time it is called. After that, the
// generated binary is used for subsequent requests.
func poetExecutablePath(baseDir string) (string, error) {
	compileMtx.Lock()
	defer compileMtx.Unlock()

	// If poet has already been compiled, just use that.
	if len(executablePath) != 0 {
		return executablePath, nil
	}

	// Build poet and output an executable in a static temp path.
	outputPath := filepath.Join(baseDir, "poet")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}

	cmd := exec.Command(
		"go", "build", "-o", outputPath, "github.com/spacemeshos/poet",
	)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to build poet: failed to get stderr: %s", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to build poet: failed to start go build: %s", err)
	}

	slurp, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to build poet: go build failed: %w\nstderr: %s", err, slurp)
	}

	// Save executable path so future calls do not recompile.
	executablePath = outputPath
	return executablePath, nil
}
