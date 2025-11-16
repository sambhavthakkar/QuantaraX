package transport

import (
	"context"
	"sync"

	"github.com/quic-go/quic-go"
)

// PriorityScheduler multiplexes chunk sends across priority classes.
type PriorityScheduler struct {
	conn   *quic.Conn
	queues map[PriorityClass]chan func(context.Context)
	wg     sync.WaitGroup
}

func NewPriorityScheduler(conn *quic.Conn) *PriorityScheduler {
	qs := &PriorityScheduler{
		conn:   conn,
		queues: map[PriorityClass]chan func(context.Context){
			PriorityP0: make(chan func(context.Context), 128),
			PriorityP1: make(chan func(context.Context), 128),
			PriorityP2: make(chan func(context.Context), 128),
		},
	}
	// Start dispatchers with weighted round-robin preference P0>P1>P2
	qs.wg.Add(1)
	go func() {
		defer qs.wg.Done()
		ctx := context.Background()
		for {
			select {
			case f, ok := <-qs.queues[PriorityP0]:
				if !ok { return }
				if f != nil { f(ctx) }
			default:
				select {
				case f, ok := <-qs.queues[PriorityP1]:
					if !ok { return }
					if f != nil { f(ctx) }
				default:
					f, ok := <-qs.queues[PriorityP2]
					if !ok { return }
					if f != nil { f(ctx) }
				}
			}
		}
	}()
	return qs
}

func (qs *PriorityScheduler) Enqueue(class PriorityClass, fn func(context.Context)) {
	qs.queues[class] <- fn
}

func (qs *PriorityScheduler) Close() {
	for _, q := range qs.queues {
		close(q)
	}
	qs.wg.Wait()
}
