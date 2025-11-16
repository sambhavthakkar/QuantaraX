package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/quantarax/backend/daemon/transport"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto"
)

// SendWithOrchestration demonstrates routing control/preview/bulk via OrchestratedSender.
// This is a scaffold for the daemon's send pipeline to call after QUIC is established.
func SendWithOrchestration(ctx context.Context, conn *transport.QUICConnection, manifest *chunker.Manifest, sessionKeys *crypto.SessionKeys, sessionID uuid.UUID, filePath string, onChunkSent func(int64)) error {
	profile := transport.ProfileForDomain(manifest.Domain, manifest)
	// onFailed enqueues DTN retry if configured
	onFailed := func(idx int64, err error) {
		if manifest.DTNProfile == nil {
			return
		}
		q := GetDTNQueue()
		if q == nil {
			return
		}
		expire := time.Now().Add(time.Duration(manifest.DTNProfile.TTLSeconds) * time.Second).Unix()
		_ = q.Enqueue(&DTNItem{SessionID: manifest.SessionID, ChunkIdx: idx, Priority: 1, ExpireAt: expire})
	}
	orch := transport.NewOrchestratedSender(conn, profile, sessionKeys, sessionID, filePath, int64(manifest.ChunkSize), onChunkSent, onFailed)
	defer orch.Close()
	// Start autotuner for chunk size and streams
	auto := transport.NewAutoTuner(orch, manifest)
	auto.Start()
	defer auto.Stop()
	// Runtime FEC adaptation using control stream
	fecCtl := transport.NewFECController(manifest.FEC.K, manifest.FEC.R, func(k, r int, reason string) {
		if conn.GetControlStream() != nil {
			_ = conn.GetControlStream().SendFECUpdate(&transport.FECUpdateMessage{SessionID: manifest.SessionID, K: k, R: r, Reason: reason, Timestamp: time.Now().Unix()})
		}
	})
	go func() {
		Ticker := time.NewTicker(5 * time.Second)
		defer Ticker.Stop()
		for range Ticker.C {
			fecCtl.Tick()
		}
	}()
	// Preflight CAS negotiation: ask receiver what chunks it has in CAS
	have := map[int64]bool{}
	if conn.GetControlStream() != nil {
		_ = conn.GetControlStream().SendChunkHaveRequest(&transport.ChunkHaveRequest{SessionID: manifest.SessionID, ChunkCount: int(manifest.ChunkCount)})
		// Best-effort receive response (non-blocking in production)
		if t, data, err := conn.GetControlStream().ReceiveAny(); err == nil && t == transport.MessageTypeChunkHaveResponse {
			var resp transport.ChunkHaveResponse
			if json.Unmarshal(data, &resp) == nil {
				var decomp transport.ChunkRangeCompressor
				idxs, _ := decomp.Decompress(resp.HaveRanges)
				for _, id := range idxs {
					have[id] = true
				}
			}
		}
	}
	// Control example
	orch.EnqueueControl(func(ctx context.Context) {
		fmt.Println("control: preflight complete")
	})
	// Spawn a control listener to handle NACK and retransmit missing chunks
	go func() {
		for {
			if conn.GetControlStream() == nil {
				return
			}
			t, data, err := conn.GetControlStream().ReceiveAny()
			if err != nil {
				return
			}
			if t == transport.MessageTypeNack {
				var nack transport.NackMessage
				if json.Unmarshal(data, &nack) == nil {
					var decomp transport.ChunkRangeCompressor
					idxs, _ := decomp.Decompress(nack.MissingRanges)
					for _, id := range idxs {
						_ = orch.EnqueueBulk(id)
					}
				}
			}
		}
	}()
	// Preview/header scheduling example (first 3 chunks)
	for i := int64(0); i < 3 && i < int64(manifest.ChunkCount); i++ {
		if have[i] {
			continue
		}
		_ = orch.EnqueuePreview(i)
	}
	// Bulk scheduling example (rest chunks)
	for i := int64(3); i < int64(manifest.ChunkCount); i++ {
		if have[i] {
			continue
		}
		_ = orch.EnqueueBulk(i)
	}
	return nil
}
