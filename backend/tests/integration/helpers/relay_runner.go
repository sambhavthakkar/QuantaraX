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

// RelayRunner manages relay service lifecycle for integration tests
type RelayRunner struct {
	BinaryPath     string
	ListenAddr     string
	MaxConnections int
	AuthMode       string
	LogPath        string
	Process        *exec.Cmd
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewRelayRunner creates a new relay runner
func NewRelayRunner(binaryPath, listenAddr string) *RelayRunner {
	return &RelayRunner{
		BinaryPath:     binaryPath,
		ListenAddr:     listenAddr,
		AuthMode:       "none",
		MaxConnections: 100,
	}
}

// Start starts the relay service
func (rr *RelayRunner) Start() error {
	rr.ctx, rr.cancel = context.WithCancel(context.Background())
	// Create log file
	logDir := filepath.Join("test-logs")
	os.MkdirAll(logDir, 0755)
	rr.LogPath = filepath.Join(logDir, fmt.Sprintf("relay-%d.log", time.Now().Unix()))
	logFile, err := os.Create(rr.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	args := []string{"--listen", rr.ListenAddr}
	if rr.MaxConnections > 0 {
		args = append(args, "--max-connections", fmt.Sprintf("%d", rr.MaxConnections))
	}
	if rr.AuthMode != "" {
		args = append(args, "--auth-mode", rr.AuthMode)
	}

	rr.Process = exec.CommandContext(rr.ctx, rr.BinaryPath, args...)
	rr.Process.Stdout = logFile
	rr.Process.Stderr = logFile

	if err := rr.Process.Start(); err != nil {
		return fmt.Errorf("failed to start relay: %w", err)
	}

	// Wait for relay to be ready by polling /health instead of fixed sleep
	readyCtx, cancel := context.WithTimeout(rr.ctx, 10*time.Second)
	defer cancel()
	for {
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("relay readiness timeout: %w", readyCtx.Err())
		default:
		}
		resp, err := http.Get(rr.GetURL() + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			break
		}
		if resp != nil { resp.Body.Close() }
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// Stop stops the relay service
func (rr *RelayRunner) Stop() error {
	if rr.cancel != nil {
		rr.cancel()
	}
	if rr.Process != nil && rr.Process.Process != nil {
		// Give it a moment to shutdown gracefully
		time.Sleep(1 * time.Second)
		if err := rr.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill relay: %w", err)
		}
	}
	return nil
}

// IsRunning checks if the relay is still running
func (rr *RelayRunner) IsRunning() bool {
	if rr.Process == nil {
		return false
	}
	return rr.Process.ProcessState == nil || !rr.Process.ProcessState.Exited()
}

// GetURL returns the relay service URL for HTTP endpoints (health/metrics)
func (rr *RelayRunner) GetURL() string {
	// Relay exposes HTTP on a fixed port :8083; reuse the host from the QUIC listen addr
	host := rr.ListenAddr
	// if listen is host:port, strip port
	if idx := len(host) - 1; idx >= 0 {
		// naive split: find last ':' and take left side as host
		for i := len(host) - 1; i >= 0; i-- {
			if host[i] == ':' {
				host = host[:i]
				break
			}
		}
	}
	if host == "" || host == ":" {
		host = "localhost"
	}
	return "http://" + host + ":8083"
}
