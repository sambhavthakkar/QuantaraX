package service

import (
	"time"
)

type DTNWorker struct {
	queue *DTNQueue
	stop  chan struct{}
	// hooks to send chunks
	sendFunc func(sess string, idx int64) error
}

func NewDTNWorker(q *DTNQueue, send func(string, int64) error) *DTNWorker {
	return &DTNWorker{queue: q, stop: make(chan struct{}), sendFunc: send}
}

func (w *DTNWorker) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ticker.C:
				items, _ := w.queue.DequeueBatch(128)
				for _, it := range items {
					_ = w.sendFunc(it.SessionID, it.ChunkIdx)
				}
			}
		}
	}()
}

func (w *DTNWorker) Stop() { close(w.stop) }
