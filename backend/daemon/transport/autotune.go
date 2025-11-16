package transport

import (
	"github.com/quantarax/backend/internal/chunker"
	"time"
)

// clamp rounds to nearest 256KiB multiple between 256KiB and 8MiB.
func clampChunkBytes(v int) int {
	if v < 256*1024 {
		v = 256 * 1024
	}
	if v > 8*1024*1024 {
		v = 8 * 1024 * 1024
	}
	// round to 256KiB multiple
	rem := v % (256 * 1024)
	if rem != 0 {
		v = v - rem + (256 * 1024)
	}
	return v
}

// computeBDPChunk estimates per-stream chunk size from RTT and bandwidth.
// Bandwidth is in Mbps (as in manifest.Network.Bandwidth), RTT in ms.
func computeBDPChunk(network *chunker.NetworkProfile, streams int) int {
	if network == nil || streams <= 0 {
		return 1024 * 1024
	}
	bwBps := network.Bandwidth * 1_000_000 / 8.0 // Mbps -> bytes/sec
	rttSec := network.RTTMsAvg / 1000.0
	bdpBytes := int(bwBps * rttSec)
	perStream := bdpBytes / streams
	if perStream <= 0 {
		perStream = 256 * 1024
	}
	return clampChunkBytes(perStream)
}

// AutoTuner periodically adjusts worker streams (8..16) and chunk sizes (256KiB..8MiB)
// based on coarse network profile. Hooks can be extended to use live metrics.
type AutoTuner struct {
	orch     *OrchestratedSender
	manifest *chunker.Manifest
	quit     chan struct{}
}

func NewAutoTuner(orch *OrchestratedSender, manifest *chunker.Manifest) *AutoTuner {
	return &AutoTuner{orch: orch, manifest: manifest, quit: make(chan struct{})}
}

func (a *AutoTuner) Start() {
	go func() {
		// Probe phase: 5s at 256KiB and 8 streams
		probeUntil := time.Now().Add(5 * time.Second)
		for time.Now().Before(probeUntil) {
			a.orch.Adjust(256*1024, 8)
			time.Sleep(500 * time.Millisecond)
		}
		// Periodic tuning
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-a.quit:
				return
			case <-ticker.C:
				streams := 8
				if a.manifest.Network != nil {
					if a.manifest.Network.Bandwidth >= 200 {
						streams = 16
					} else if a.manifest.Network.Bandwidth >= 80 {
						streams = 12
					}
				}
				chunkBytes := computeBDPChunk(a.manifest.Network, streams)
				a.orch.Adjust(chunkBytes, streams)
			}
		}
	}()
}

func (a *AutoTuner) Stop() { close(a.quit) }
