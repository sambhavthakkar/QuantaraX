package transport

import (
	"context"

	"path/filepath"

	"github.com/google/uuid"
	"github.com/quantarax/backend/internal/crypto"
)

// OrchestratedSender manages per-class worker pools and control routing.
type OrchestratedSender struct {
	conn   *QUICConnection
	pools  map[PriorityClass]*ChunkWorkerPool
}

// NewOrchestratedSender creates worker pools for P0/P1/P2 according to the domain profile.
func NewOrchestratedSender(conn *QUICConnection, profile DomainTransportProfile, sessionKeys *crypto.SessionKeys, sessionID uuid.UUID, filePath string, baseChunkSize int64, onChunkSent func(idx int64), onChunkFailed func(idx int64, err error)) *OrchestratedSender {
	pools := make(map[PriorityClass]*ChunkWorkerPool)
	mk := func(class PriorityClass, cfg ClassConfig) *ChunkWorkerPool {
		workers := cfg.Streams
		if workers <= 0 { workers = 1 }
		chunkSize := baseChunkSize
		if cfg.ChunkBytes > 0 { chunkSize = int64(cfg.ChunkBytes) }
	p := NewChunkWorkerPool(workers, 1024, conn.GetConnection(), sessionKeys, sessionID, filePath, chunkSize, onChunkSent, onChunkFailed)
		p.SetScheduler(conn.Scheduler(), class)
		return p
	}
	// Override class chunk sizes with BDP-based sizing where network info available
	pools[PriorityP0] = mk(PriorityP0, profile.P0)
	pools[PriorityP1] = mk(PriorityP1, profile.P1)
	pools[PriorityP2] = mk(PriorityP2, profile.P2)
	// Start pools
	for _, p := range pools { p.Start() }
	return &OrchestratedSender{conn: conn, pools: pools}
}

// EnqueueControl schedules a control task on P0.
func (s *OrchestratedSender) EnqueueControl(fn func(context.Context)) {
	s.conn.Scheduler().Enqueue(PriorityP0, fn)
}

// EnqueuePreview schedules a chunk index on P1 for headers/preview frames.
func (s *OrchestratedSender) EnqueuePreview(chunkIndex int64) error {
	return s.pools[PriorityP1].EnqueueChunk(chunkIndex)
}

// EnqueueBulk schedules a chunk index on P2 for bulk payload.
func (s *OrchestratedSender) EnqueueBulk(chunkIndex int64) error {
	return s.pools[PriorityP2].EnqueueChunk(chunkIndex)
}

// Close stops all pools.
func (s *OrchestratedSender) Close() {
	for _, p := range s.pools { p.Stop() }
}

// Adjust updates chunk sizes and worker counts according to autotuning decisions.
func (s *OrchestratedSender) Adjust(chunkBytes int, totalStreams int) {
	if totalStreams < 2 { totalStreams = 2 }
	p1 := totalStreams / 2
	p2 := totalStreams - p1
	if pool, ok := s.pools[PriorityP1]; ok {
		pool.SetChunkSize(int64(chunkBytes))
		pool.ScaleWorkers(p1)
	}
	if pool, ok := s.pools[PriorityP2]; ok {
		pool.SetChunkSize(int64(chunkBytes))
		pool.ScaleWorkers(p2)
	}
}

// Helper to derive a preview path near file (optional usage)
func previewPathFor(filePath string) string {
	base := filepath.Base(filePath)
	return base + ".preview.jpg"
}
