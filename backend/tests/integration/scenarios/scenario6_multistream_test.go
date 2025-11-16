package scenarios

import (
	"crypto/ed25519"
	"encoding/json"
	"github.com/quic-go/quic-go"
	"context"
	"fmt"
	"testing"
	"time"
	"path/filepath"
	"os"

	"github.com/google/uuid"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/quicutil"
	"github.com/quantarax/backend/daemon/transport"
	"github.com/quantarax/backend/tests/integration/helpers"
)

// TestScenario6MultiStream orchestrates a true multi-stream transfer using OrchestratedSender
// and the QUIC control stream between a lightweight sender/receiver harness.
func TestScenario6MultiStream(t *testing.T) {
	t.Log("=== Scenario 6: QUIC Multi-Stream (Orchestrated) ===")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Generate a temporary 32MB file
	fileGen, err := helpers.NewFileGenerator()
	if err != nil { t.Fatalf("file gen: %v", err) }
	defer fileGen.Cleanup()
	filePath, _, err := fileGen.GenerateFile("scenario6-16mb.bin", 16*1024*1024)
	if err != nil { t.Fatalf("gen: %v", err) }

	// Build manifest
	mf, err := chunker.ComputeManifest(filePath, chunker.ChunkOptions{ChunkSize: 1<<20})
	if err != nil { t.Fatalf("manifest: %v", err) }
	mf.Domain = "media" // enables a higher stream count in profile
	mf.SessionID = uuid.New().String()

	// Deterministic session keys (demo style)
	var theirPubKey [32]byte
	var manifestHash [32]byte
	for i := range theirPubKey { theirPubKey[i] = 0x11 }
	for i := range manifestHash { manifestHash[i] = 0x22 }
	kp, _ := crypto.GenerateX25519()
	sessionKeys, err := crypto.DeriveSessionKeys(&kp.PrivateKey, &theirPubKey, manifestHash[:])
	if err != nil { t.Fatalf("derive keys: %v", err) }

	// Prepare TLS
	cert, key, err := quicutil.GenerateSelfSignedCert()
	if err != nil { t.Fatalf("tls cert: %v", err) }
	tlsServer, err := quicutil.MakeTLSConfig(cert, key)
	if err != nil { t.Fatalf("tls server: %v", err) }
	tlsServer.NextProtos = []string{"quantarax-quic"}
	tlsClient := quicutil.MakeClientTLSConfig()
	tlsClient.NextProtos = []string{"quantarax-quic"}

	// Allocate a free port and start receiver
	port, err := helpers.GetFreeUDPPort()
	if err != nil { t.Fatalf("port: %v", err) }
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := transport.ListenQUIC(addr, tlsServer)
	if err != nil { t.Fatalf("listen: %v", err) }
	defer listener.Close()

	recvDir := fileGen.MakeTempDir("recv")
	outputPath := filepath.Join(recvDir, filepath.Base(filePath))

	// Receiver goroutine
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil { done <- fmt.Errorf("accept: %w", err); return }
		ctrl, err := conn.AcceptControlStream(ctx)
		if err != nil { done <- fmt.Errorf("accept control: %w", err); return }
		// Receive signed manifest and parse
		signed, err := ctrl.ReceiveSignedManifest()
		if err != nil { done <- fmt.Errorf("recv manifest: %w", err); return }
		var rmf chunker.Manifest
		if err := json.Unmarshal(signed.ManifestJSON, &rmf); err != nil { done <- fmt.Errorf("parse manifest json: %w", err); return }
		// Build receiver ChunkReceiver
		sid, _ := uuid.Parse(rmf.SessionID)
		r := transport.NewChunkReceiver(conn.GetConnection(), sessionKeys, sid, outputPath, int64(rmf.ChunkSize), nil, ctrl, &rmf, nil, nil)
		go r.AcceptAndProcessStreams()
		// Control loop: handle CAS preflight only; verification will be received by sender
		go func(){
			for {
				t, data, err := ctrl.ReceiveAny()
				if err != nil { return }
				if t == transport.MessageTypeChunkHaveRequest {
					var req transport.ChunkHaveRequest
					if json.Unmarshal(data, &req) == nil {
						_ = ctrl.SendChunkHaveResponse(&transport.ChunkHaveResponse{SessionID: req.SessionID, HaveRanges: "", ChunkCount: req.ChunkCount, Timestamp: time.Now().Unix()})
					}
				}
			}
		}()
		// Keep receiver alive until test context is cancelled
		<-ctx.Done()
		_ = conn.Close()
		done <- nil
	}()

	// Sender path: dial, open control, send signed manifest, then orchestrated send
	clientConn, err := quic.DialAddr(ctx, addr, tlsClient, &quic.Config{EnableDatagrams: false})
	if err != nil { t.Fatalf("dial: %v", err) }
	defer clientConn.CloseWithError(0, "done")
	qc := transport.NewQUICConnection(clientConn)
	ctrl, err := qc.OpenControlStream(ctx)
	if err != nil { t.Fatalf("open control: %v", err) }
	// Sign manifest with a throwaway ed25519
	_, edPriv, _ := ed25519.GenerateKey(nil)
	mfBytes, _ := json.Marshal(mf)
	if err := ctrl.SendSignedManifest(mfBytes, edPriv); err != nil { t.Fatalf("send manifest: %v", err) }
	// Invoke orchestrated sending
	sid, _ := uuid.Parse(mf.SessionID)
	onSent := func(idx int64) {}
	if err := helpers.SendWithOrchestration(ctx, qc, mf, sessionKeys, sid, filePath, onSent); err != nil {
		t.Fatalf("orchestrated send: %v", err)
	}

	// Sender verified transfer; signal receiver to stop and wait briefly for shutdown
	cancel()
	select {
	case err := <-done:
		if err != nil { t.Fatalf("receiver: %v", err) }
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for receiver shutdown")
	}

	// Ensure output file exists and is non-empty
	st, err := os.Stat(outputPath)
	if err != nil { t.Fatalf("output: %v", err) }
	if st.Size() == 0 { t.Fatalf("output empty") }
}
