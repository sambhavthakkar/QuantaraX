package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/chunker"
	"encoding/json"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/quantarax/backend/daemon/config"
	"github.com/quantarax/backend/daemon/manager"
	"github.com/quantarax/backend/daemon/service"
	"github.com/quantarax/backend/daemon/transport"
	"github.com/quantarax/backend/internal/observability"
	"github.com/quantarax/backend/internal/quicutil"
	"github.com/quantarax/backend/internal/ratelimit"
)

func main() {
	// Initialize observability
	logger := observability.NewLogger("quantarax-daemon", "1.0.0", os.Stdout)
	// Initialize CAS backend (in-memory; replace with Bolt in production)
	service.InitCAS()
	// Start periodic CAS GC when BoltCAS is used (24h retention, hourly interval)
	service.StartCASGCLoop(24*time.Hour, 1*time.Hour)
	// Initialize DTN queue/worker
	_ = service.InitDTN("/tmp/quantarax_dtn.db")
	metrics := observability.NewMetrics()
	healthChecker := observability.NewHealthChecker("1.0.0")
	// Init tracing if configured
	if shutdown, err := observability.InitTracing(context.Background(), "quantarax-daemon"); err == nil { defer shutdown(context.Background()) }
	
	logger.Info("QuantaraX Daemon starting...")
	
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		logger.Fatal(err, "Failed to load config")
	}
	
	logger.Info("Configuration loaded")
	log.Printf("  QUIC Address: %s", cfg.QUICAddress)
	log.Printf("  Chunk Size: %d bytes", cfg.ChunkSize)
	log.Printf("  Worker Count: %d", cfg.WorkerCount)
	
	// Initialize session store
	sessionStore := manager.NewSessionStore()
	logger.Info("Session store initialized")
	
	// Initialize event publisher
	eventPublisher := service.NewEventPublisher(cfg.EventBufferSize)
	log.Printf("Event publisher initialized (buffer size: %d)", cfg.EventBufferSize)
	
	// Initialize transfer service
	transferService, err := service.NewTransferService(
		sessionStore,
		eventPublisher,
		cfg.KeysDirectory,
		cfg.ChunkSize,
	)
	if err != nil {
		logger.Fatal(err, "Failed to initialize transfer service")
	}
	logger.Info("Transfer service initialized")
	
	// Register health checks
	healthChecker.RegisterCheck("quic_listener", observability.QUICListenerCheck(cfg.QUICAddress))
	healthChecker.RegisterCheck("keystore", observability.KeystoreCheck(true))
	healthChecker.RegisterCheck("database", observability.DatabaseCheck("./data/quantarax.db"))
	
	// Generate self-signed TLS certificate for QUIC
	certPEM, keyPEM, err := quicutil.GenerateSelfSignedCert()
	if err != nil {
		logger.Fatal(err, "Failed to generate TLS certificate")
	}
	logger.Info("Generated self-signed TLS certificate for QUIC")
	
	tlsConfig, err := quicutil.MakeTLSConfig(certPEM, keyPEM)
	if err != nil {
		logger.Fatal(err, "Failed to create TLS config")
	}
	
	// Start QUIC listener
	quicListener, err := transport.ListenQUIC(cfg.QUICAddress, tlsConfig)
	if err != nil {
		logger.Fatal(err, "Failed to start QUIC listener")
	}
	defer quicListener.Close()
	
	logger.Info("QUIC listener started on " + cfg.QUICAddress)
	
	// Start metrics and health HTTP server
go startObservabilityServer(metrics, healthChecker, logger) // exposes /metrics, /health, /debug/pprof
	
	// Start accepting QUIC connections in background
ctx, cancel := context.WithCancel(context.Background())
	// Rate limiter: limit new connections per second
	tb := ratelimit.NewTokenBucket(50, 100) // 50 conn/s, burst 100
	defer cancel()
	
go func() { // connection accept loop (rate-limited)
		for {
			select {
			case <-ctx.Done():
				return
			default:
// rate limit connection accepts
			if !tb.Allow(1) { time.Sleep(10 * time.Millisecond); continue }
			conn, err := quicListener.Accept(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					logger.Error(err, "Failed to accept QUIC connection")
					metrics.RecordQUICConnection(false)
					continue
				}
				
				logger.ConnectionEstablished(conn.GetConnection().RemoteAddr().String(), "conn-id")
				metrics.RecordQUICConnection(true)
				
				// Handle connection in goroutine
				go handleConnection(ctx, conn, transferService, eventPublisher, cfg, logger, metrics)
			}
		}
	}()
	
	logger.Info("QuantaraX Daemon running")
	logger.Info("Press Ctrl+C to stop")
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	logger.Info("Shutting down gracefully...")
	cancel()
	
	// Cleanup old sessions
	cleanedUp := sessionStore.CleanupOldSessions(24 * time.Hour)
	log.Printf("Cleaned up %d old sessions", cleanedUp)
	
	logger.Info("Daemon stopped")
}

func startObservabilityServer(metrics *observability.Metrics, health *observability.HealthChecker, logger *observability.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/health", health.Handler())
	// pprof endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{Addr: ":8081", Handler: mux}
	logger.Info("Observability server listening on :8081 (metrics, health, pprof)")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err, "Observability server error")
	}
}

func handleConnection(
	ctx context.Context,
	conn *transport.QUICConnection,
	transferService *service.TransferService,
	eventPublisher *service.EventPublisher,
	cfg *config.Config,
	logger *observability.Logger,
	metrics *observability.Metrics,
) {
	defer conn.Close()
	
	// Accept control stream and receive signed manifest
	ctrl, err := conn.AcceptControlStream(ctx)
	if err != nil { logger.Error(err, "failed to accept control stream"); return }
	signed, err := ctrl.ReceiveSignedManifest()
	if err != nil { logger.Error(err, "failed to receive manifest"); return }
	logger.Info("Manifest received")
	// Parse manifest JSON
	var manifest chunker.Manifest
	if err := json.Unmarshal(signed.ManifestJSON, &manifest); err != nil {
		logger.Error(err, "failed to parse manifest JSON"); return
	}
	// Build basic session keys placeholder (real key exchange omitted here)
	var sk crypto.SessionKeys
	// Orchestrate sending using domain profile
	_ = transport.ProfileForDomain(manifest.Domain, &manifest)
	// NOTE: file path unknown here; in a full pipeline, resolve mapped storage path
	filePath := manifest.FileName
	sessionUUID, _ := uuid.Parse(manifest.SessionID)
	onChunkSent := func(idx int64) { metrics.RecordChunkSent(int(idx)) }
	if err := service.SendWithOrchestration(ctx, conn, &manifest, &sk, sessionUUID, filePath, onChunkSent); err != nil {
		logger.Error(err, "send orchestration failed"); return
	}
	logger.Info("Orchestrated transfer scheduled")
}
