package helpers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DaemonRunner manages the lifecycle of the QuantaraX daemon for tests.
type DaemonRunner struct {
	BinaryPath string
	ListenAddr string
	GRPCAddr   string
	RESTAddr   string
	ObservAddr string
	LogPath    string
	Process    *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewDaemonRunner creates a new daemon runner.
func NewDaemonRunner(binaryPath, grpcAddr, restAddr, observAddr string) *DaemonRunner {
	return &DaemonRunner{
		BinaryPath: binaryPath,
		ListenAddr: grpcAddr, // Use grpcAddr as primary
		GRPCAddr:   grpcAddr,
		RESTAddr:   restAddr,
		ObservAddr: observAddr,
	}
}

// Start launches the daemon process and waits for its /health endpoint to be ready.
func (r *DaemonRunner) Start() error {
	r.ctx, r.cancel = context.WithCancel(context.Background())

	logDir := filepath.Join("test-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	r.LogPath = filepath.Join(logDir, fmt.Sprintf("daemon-%d.log", time.Now().Unix()))

	logFile, err := os.Create(r.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	r.Process = exec.CommandContext(
		r.ctx,
		r.BinaryPath,
		"--grpc-addr", ":9090",
		"--rest-addr", ":18080",
		"--observ-addr", ":8081",
		"--mode", "test",
	)
	r.Process.Stdout = logFile
	r.Process.Stderr = logFile

	if err := r.Process.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon health endpoint
	if err := r.waitForReady(); err != nil {
		return fmt.Errorf("daemon did not become ready: %w", err)
	}
	return nil
}

// waitForReady polls /health until the daemon reports ready.
func (r *DaemonRunner) waitForReady() error {
	url := "http://127.0.0.1:8081/health"
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
	}
	return fmt.Errorf("daemon not ready after timeout")
}

// Stop terminates the daemon.
func (r *DaemonRunner) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.Process != nil && r.Process.Process != nil {
		time.Sleep(200 * time.Millisecond)
		if err := r.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
	}
	return nil
}
