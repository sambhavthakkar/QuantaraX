package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BootstrapRunner manages the bootstrap service lifecycle for integration tests.
type BootstrapRunner struct {
	BinaryPath string
	ListenAddr string
	LogPath    string
	Process    *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewBootstrapRunner creates a new bootstrap runner instance.
func NewBootstrapRunner(binaryPath, listenAddr string) *BootstrapRunner {
	return &BootstrapRunner{
		BinaryPath: binaryPath,
		ListenAddr: listenAddr,
	}
}

// Start launches the bootstrap service and begins logging output.
func (br *BootstrapRunner) Start() error {
	br.ctx, br.cancel = context.WithCancel(context.Background())

	// Create test logs directory if it doesn’t exist
	logDir := filepath.Join("test-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	br.LogPath = filepath.Join(logDir, fmt.Sprintf("bootstrap-%d.log", time.Now().Unix()))

	logFile, err := os.Create(br.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Start bootstrap process
	br.Process = exec.CommandContext(br.ctx, br.BinaryPath, "--listen", br.ListenAddr)
	br.Process.Stdout = logFile
	br.Process.Stderr = logFile

	if err := br.Process.Start(); err != nil {
		return fmt.Errorf("failed to start bootstrap: %w", err)
	}

	// Wait briefly for the bootstrap service to initialize
	time.Sleep(2 * time.Second)

	return nil
}

// Stop gracefully stops the bootstrap service.
func (br *BootstrapRunner) Stop() error {
	if br.cancel != nil {
		br.cancel()
	}

	if br.Process != nil && br.Process.Process != nil {
		time.Sleep(1 * time.Second)
		if err := br.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill bootstrap: %w", err)
		}
	}

	return nil
}

// GetURL returns the full HTTP URL for the bootstrap service.
func (br *BootstrapRunner) GetURL() string {
	return "http://" + br.ListenAddr
}
