package helpers

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// GetFreeUDPPort finds an available UDP port on localhost and returns it.
func GetFreeUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.LocalAddr().(*net.UDPAddr).Port, nil
}

// QuicRecvRunner manages the lifecycle of a QUIC receiver process during tests.
type QuicRecvRunner struct {
	BinaryPath string
	Listen     string
	OutputDir  string
	LogPath    string
	Process    *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewQuicRecvRunner creates a new instance of QuicRecvRunner.
func NewQuicRecvRunner(binaryPath, listen, outputDir string) *QuicRecvRunner {
	return &QuicRecvRunner{
		BinaryPath: binaryPath,
		Listen:     listen,
		OutputDir:  outputDir,
	}
}

// Start launches the QUIC receiver process and logs output to a file.
func (r *QuicRecvRunner) Start() error {
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// Ensure test log directory exists
	logDir := filepath.Join("test-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	r.LogPath = filepath.Join(logDir, fmt.Sprintf("quic-recv-%d.log", time.Now().Unix()))
	logFile, err := os.Create(r.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Build and start the QUIC receiver command
	r.Process = exec.CommandContext(
		r.ctx,
		r.BinaryPath,
		"--listen", r.Listen,
		"--output-dir", r.OutputDir,
	)
	r.Process.Stdout = logFile
	r.Process.Stderr = logFile

	if err := r.Process.Start(); err != nil {
		return fmt.Errorf("failed to start quic_recv: %w", err)
	}

	// Give it a short time to start up
	time.Sleep(1 * time.Second)
	return nil
}

// Stop gracefully terminates the QUIC receiver process.
func (r *QuicRecvRunner) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}

	if r.Process != nil && r.Process.Process != nil {
		time.Sleep(200 * time.Millisecond)
		if err := r.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill quic_recv: %w", err)
		}
	}

	return nil
}

// RunQuicSend executes a QUIC sender process synchronously with the given arguments.
func RunQuicSend(binaryPath string, args ...string) error {
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
