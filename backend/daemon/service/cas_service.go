package service

import (
	"time"
	"path/filepath"
	"os"
	"github.com/quantarax/backend/daemon/transport"
	"github.com/quantarax/backend/daemon/manager"
)

var defaultDTNQueue *DTNQueue
var boltCAS *manager.BoltCAS

func InitDTN(path string) error {
	q, err := OpenDTNQueue(path)
	if err != nil { return err }
	defaultDTNQueue = q
	w := NewDTNWorker(q, func(sess string, idx int64) error {
		// TODO: signal transfer service to retry sending this chunk if connection available
		return nil
	})
	w.Start()
	return nil
}

func GetDTNQueue() *DTNQueue { return defaultDTNQueue }

// Bolt-backed CAS with periodic GC

type InMemoryCAS struct { m map[string]time.Time }
func NewInMemoryCAS() *InMemoryCAS { return &InMemoryCAS{m: make(map[string]time.Time)} }
func (c *InMemoryCAS) HasChunk(hash string) bool { _, ok := c.m[hash]; return ok }
func (c *InMemoryCAS) PutChunk(hash string, length int) error { c.m[hash] = time.Now(); return nil }

// InitCAS initializes the CAS backend; prefer BoltCAS under ~/.local/share/quantarax/cas.db and fallback to in-memory.
func InitCAS() {
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".local", "share", "quantarax", "cas.db")
	_ = os.MkdirAll(filepath.Dir(defaultPath), 0o755)
	if bc, err := manager.OpenBoltCAS(defaultPath); err == nil {
		boltCAS = bc
		transport.SetCASBackend(boltCAS)
	} else {
		transport.SetCASBackend(NewInMemoryCAS())
	}
}

// StartCASGCLoop starts a periodic GC loop for BoltCAS; no-op for in-memory.
func StartCASGCLoop(retention time.Duration, interval time.Duration) {
	if boltCAS == nil { return }
	go func(){
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = boltCAS.GC(retention)
		}
	}()
}
