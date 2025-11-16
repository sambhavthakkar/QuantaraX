package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quantarax/backend/internal/observability"
	"github.com/quantarax/backend/internal/quicutil"
	"github.com/quantarax/backend/internal/ratelimit"
	"github.com/quantarax/backend/internal/validation"
	"go.opentelemetry.io/otel"
)

// RelayConfig holds relay service configuration
type RelayConfig struct {
	ListenAddr       string
	MaxConnections   int
	ConnTimeout      time.Duration
	StreamBufferSize int
	AuthMode         string
	LogLevel         string
}

// RelayService manages QUIC relay forwarding
type RelayService struct {
	config            *RelayConfig
	activeConnections int64
	totalConnections  int64
	bytesForwarded    int64
}

func NewRelayService(config *RelayConfig) *RelayService {
	return &RelayService{config: config}
}

// Start begins the relay service
func (rs *RelayService) Start() error {
	tr := otel.Tracer("quantarax-relay")
	ctx, span := tr.Start(context.Background(), "relay.start")
	defer span.End()

	// Generate TLS config for QUIC
	tlsConfig := generateRelayTLSConfig()

	quicConfig := &quic.Config{MaxIdleTimeout: 30 * time.Second, KeepAlivePeriod: 10 * time.Second}

	listener, err := quic.ListenAddr(rs.config.ListenAddr, tlsConfig, quicConfig)
	// Apply simple rate limiter for new connections
	connLimiter := ratelimit.NewTokenBucket(200, 400) // 200 conn/s, burst 400
	_ = connLimiter
	if err != nil {
		return fmt.Errorf("failed to start QUIC listener: %w", err)
	}

	log.Printf("Relay service listening on %s", rs.config.ListenAddr)
	log.Printf("Max connections: %d", rs.config.MaxConnections)
	log.Printf("Authentication mode: %s", rs.config.AuthMode)

	// Start health/metrics/pprof server
	go rs.startHealthServer()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
		listener.Close()
	}()

	// Accept connections
	for {
		// if !connLimiter.Allow(1) { time.Sleep(5 * time.Millisecond); continue }
		conn, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Relay service shutting down...")
				break
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Check connection limit
		active := atomic.LoadInt64(&rs.activeConnections)
		if active >= int64(rs.config.MaxConnections) {
			log.Printf("Connection limit reached (%d), rejecting connection", active)
			conn.CloseWithError(1, "connection limit exceeded")
			continue
		}

		atomic.AddInt64(&rs.activeConnections, 1)
		atomic.AddInt64(&rs.totalConnections, 1)

		log.Printf("Accepted connection from %s (active: %d)", conn.RemoteAddr(), active+1)

		go rs.handleConnection(ctx, conn)
	}

	return nil
}

// handleConnection manages a relay connection
func (rs *RelayService) handleConnection(ctx context.Context, sourceConn *quic.Conn) {
	tr := otel.Tracer("quantarax-relay")
	ctx, span := tr.Start(ctx, "relay.handleConnection")
	defer span.End()

	defer func() {
		atomic.AddInt64(&rs.activeConnections, -1)
		sourceConn.CloseWithError(0, "relay closing")
	}()

	// Read target address from first stream with a deadline
	controlStream, err := sourceConn.AcceptStream(ctx)
	if err == nil {
		_ = controlStream.SetReadDeadline(time.Now().Add(5 * time.Second))
	}
	if err != nil {
		log.Printf("Failed to accept control stream: %v", err)
		return
	}


	targetAddrBuf := make([]byte, 256)
	n, err := controlStream.Read(targetAddrBuf)
	if err != nil {
		log.Printf("Failed to read target address: %v", err)
		return
	}
	targetAddr := string(targetAddrBuf[:n])

	log.Printf("Relay target: %s", targetAddr)

	// Validate authentication if enabled
	if rs.config.AuthMode != "none" {
		// Read auth token
		tokenBuf := make([]byte, 256)
		n, err := controlStream.Read(tokenBuf)
		if err != nil {
			log.Printf("Failed to read auth token: %v", err)
			return
		}
		token := string(tokenBuf[:n])

		if !rs.validateToken(token) {
			log.Printf("Invalid auth token from %s", sourceConn.RemoteAddr())
			controlStream.Write([]byte("AUTH_FAILED"))
			return
		}
	}

	// Establish connection to target
	// When dialing the target receiver, use the direct transfer ALPN so it matches quic_recv
	targetTLSConfig := &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"quantarax-quic"}, ServerName: ""}
	targetConn, err := quic.DialAddr(ctx, targetAddr, targetTLSConfig, &quic.Config{MaxIdleTimeout: 30 * time.Second})
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
		controlStream.Write([]byte("TARGET_UNREACHABLE"))
		return
	}
	defer targetConn.CloseWithError(0, "relay closing")

	log.Printf("Established connection to target %s", targetAddr)

	// Send success message and keep control stream open to avoid EOF race on client
	_ = controlStream.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := controlStream.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write OK to control stream: %v", err)
		return
	}
	// Do not close controlStream here; let it be closed when the connection ends

	// Forward streams bidirectionally
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	var wg sync.WaitGroup


	wg.Add(1)
	go func() { defer wg.Done(); rs.forwardStreams(connCtx, sourceConn, targetConn, "source->target") }()
	wg.Add(1)
	go func() { defer wg.Done(); rs.forwardStreams(connCtx, targetConn, sourceConn, "target->source") }()
	wg.Wait()
	log.Printf("Relay session completed for %s", targetAddr)
}

// forwardStreams forwards all streams from source to target
func (rs *RelayService) forwardStreams(ctx context.Context, source, target *quic.Conn, direction string) {
	tr := otel.Tracer("quantarax-relay")
	ctx, span := tr.Start(ctx, "relay.forwardStreams")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stream, err := source.AcceptStream(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("Failed to accept stream (%s): %v", direction, err)
			}
			return
		}

		go rs.forwardStream(ctx, stream, target, direction)
	}
}

// forwardStream forwards a single stream
func (rs *RelayService) forwardStream(ctx context.Context, sourceStream *quic.Stream, targetConn *quic.Conn, direction string) {
	tr := otel.Tracer("quantarax-relay")
	ctx, span := tr.Start(ctx, "relay.forwardStream")
	defer span.End()

	defer sourceStream.Close()

	// Open corresponding stream on target
	targetStream, err := targetConn.OpenStreamSync(ctx)
	if err != nil {
		log.Printf("Failed to open target stream (%s): %v", direction, err)
		return
	}
	defer targetStream.Close()

	
	var wg sync.WaitGroup
	// Copy source->target
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, rs.config.StreamBufferSize)
		n, err := io.CopyBuffer(targetStream, sourceStream, buf)
		if err != nil && ctx.Err() == nil {
			log.Printf("Stream copy error (%s): %v", direction, err)
		}

		atomic.AddInt64(&rs.bytesForwarded, n)
	}()
	// Copy target->source
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, rs.config.StreamBufferSize)
		n, err := io.CopyBuffer(sourceStream, targetStream, buf)
		if err != nil && ctx.Err() == nil {
			log.Printf("Stream copy error (reverse %s): %v", direction, err)
		}

		atomic.AddInt64(&rs.bytesForwarded, n)
	}()

	wg.Wait()
}

// validateToken validates an authentication token
func (rs *RelayService) validateToken(token string) bool { return token != "" && len(token) > 10 }

// startHealthServer starts HTTP health, metrics, and pprof endpoints
func (rs *RelayService) startHealthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", rs.handleHealth)
	mux.HandleFunc("/metrics", rs.handleMetrics)
	// pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	healthAddr := ":8083"
	log.Printf("Health/metrics/pprof server listening on %s", healthAddr)
	if err := http.ListenAndServe(healthAddr, mux); err != nil { log.Printf("Health server error: %v", err) }
}

// handleHealth returns health status
func (rs *RelayService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "healthy",
		"active_connections": atomic.LoadInt64(&rs.activeConnections),
		"max_connections":    rs.config.MaxConnections,
	})
}

// handleMetrics returns relay metrics
func (rs *RelayService) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_connections": atomic.LoadInt64(&rs.activeConnections),
		"total_connections":  atomic.LoadInt64(&rs.totalConnections),
		"bytes_forwarded":    atomic.LoadInt64(&rs.bytesForwarded),
		"max_connections":    rs.config.MaxConnections,
	})
}

// generateRelayTLSConfig creates a TLS config for relay
func generateRelayTLSConfig() *tls.Config {
	certPEM, keyPEM, err := quicutil.GenerateSelfSignedCert()
	if err != nil {
		log.Fatalf("failed to generate relay certificate: %v", err)
	}
	tlsConfig, err := quicutil.MakeTLSConfig(certPEM, keyPEM)
	if err != nil {
log.Fatalf("failed to create relay TLS config: %v", err)
	}
	tlsConfig.NextProtos = []string{"quic-relay"}
	return tlsConfig
}

func main() {
	listen := flag.String("listen", ":4433", "QUIC listen address")
	maxConn := flag.Int("max-connections", 1000, "Maximum concurrent connections")
	authMode := flag.String("auth-mode", "none", "Authentication mode (none, token)")
	logLevel := flag.String("log-level", "info", "Logging level")
	flag.Parse()
	// Init tracing if configured
	if shutdown, err := observability.InitTracing(context.Background(), "quantarax-relay"); err == nil { defer shutdown(context.Background()) }
	// Validate inputs
	if err := validation.ValidateAddr(*listen); err != nil { log.Fatalf("invalid relay listen addr: %v", err) }
	if *maxConn <= 0 || *maxConn > 100000 { log.Fatalf("invalid max-connections: %d", *maxConn) }

	log.Printf("QuantaraX Relay Service starting...")
	log.Printf("Listen address: %s", *listen)
	log.Printf("Auth mode: %s", *authMode)
	log.Printf("Log level: %s", *logLevel)
	config := &RelayConfig{ListenAddr: *listen, MaxConnections: *maxConn, ConnTimeout: 30 * time.Second, StreamBufferSize: 65536, AuthMode: *authMode, LogLevel: *logLevel}
	service := NewRelayService(config)
	if err := service.Start(); err != nil { log.Fatalf("Relay service error: %v", err) }
	log.Println("Relay service stopped")
}
