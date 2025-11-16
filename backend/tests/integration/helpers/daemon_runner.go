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

	// Get a unique QUIC port for testing
	quicPort, err := GetFreeUDPPort()
	if err != nil {
		return fmt.Errorf("failed to get free UDP port: %w", err)
	}
	
	r.Process = exec.CommandContext(
		r.ctx,
		r.BinaryPath,
		"--grpc-addr", r.GRPCAddr,
		"--rest-addr", r.RESTAddr,
		"--observ-addr", r.ObservAddr,
		"--quic-addr", fmt.Sprintf(":%d", quicPort),
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
	// Extract port from ObservAddr for health check
	url := "http://" + r.ObservAddr + "/health"
	for i := 0; i < 30; i++ { // Increased attempts
		time.Sleep(1 * time.Second) // Increased delay
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		// Log progress for debugging
		if i%5 == 0 {
			fmt.Printf("Waiting for daemon health check... attempt %d/30 (url: %s)\n", i+1, url)
		}
	}
	return fmt.Errorf("daemon not ready after timeout at url: %s", url)
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
