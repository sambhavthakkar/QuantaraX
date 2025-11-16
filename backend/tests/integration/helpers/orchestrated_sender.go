package helpers

import (
	"context"
	"fmt"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/quantarax/backend/daemon/transport"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto"
)

// SendWithOrchestration is a test-local helper to drive OrchestratedSender without importing daemon/service.
func SendWithOrchestration(ctx context.Context, conn *transport.QUICConnection, manifest *chunker.Manifest, sessionKeys *crypto.SessionKeys, sessionID uuid.UUID, filePath string, onChunkSent func(int64)) error {
	profile := transport.ProfileForDomain(manifest.Domain, manifest)
	// Force fixed chunk size to match manifest for this test
	profile.P0.ChunkBytes = 0
	profile.P1.ChunkBytes = 0
	profile.P2.ChunkBytes = 0
	// Set a fixed number of streams to exercise multi-stream without autotune
	if profile.P1.Streams == 0 { profile.P1.Streams = 6 }
	if profile.P2.Streams == 0 { profile.P2.Streams = 10 }
	orch := transport.NewOrchestratedSender(conn, profile, sessionKeys, sessionID, filePath, int64(manifest.ChunkSize), onChunkSent, func(int64, error){})
	defer orch.Close()
	// Do not start autotuner in this test to keep chunk sizing stable
	// Preflight: ask receiver what chunks it has in CAS
	have := map[int64]bool{}
	if conn.GetControlStream() != nil {
		_ = conn.GetControlStream().SendChunkHaveRequest(&transport.ChunkHaveRequest{SessionID: manifest.SessionID, ChunkCount: int(manifest.ChunkCount)})
		if t, data, err := conn.GetControlStream().ReceiveAny(); err == nil && t == transport.MessageTypeChunkHaveResponse {
			var resp transport.ChunkHaveResponse
			if json.Unmarshal(data, &resp) == nil {
				var decomp transport.ChunkRangeCompressor
				idxs, _ := decomp.Decompress(resp.HaveRanges)
				for _, id := range idxs { have[id] = true }
			}
		}
	}
	// Enqueue preview and bulk
	for i := int64(0); i < 3 && i < int64(manifest.ChunkCount); i++ { if have[i] { continue }; _ = orch.EnqueuePreview(i) }
	for i := int64(3); i < int64(manifest.ChunkCount); i++ { if have[i] { continue }; _ = orch.EnqueueBulk(i) }
	// Wait for verification from receiver on the same control stream
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return context.DeadlineExceeded
		default:
			if conn.GetControlStream() == nil { time.Sleep(50*time.Millisecond); continue }
			t, data, err := conn.GetControlStream().ReceiveAny()
			if err != nil { return err }
			if t == transport.MessageTypeVerification {
				var v transport.VerificationMessage
				if json.Unmarshal(data, &v) == nil && v.Status == "SUCCESS" { return nil }
				return fmt.Errorf("verification failed or malformed")
			}
		}
	}
}

