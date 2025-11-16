package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quic-go/quic-go"
)

var (
	ErrWorkerPoolStopped = errors.New("worker pool stopped")
)

// ChunkWorkerPool manages parallel chunk transmission
type ChunkWorkerPool struct {
	workerCount   int
	chunkQueue    chan int64
	connection    *quic.Conn
	scheduler     *PriorityScheduler
	class         PriorityClass
	sessionKeys   *crypto.SessionKeys
	sessionID     uuid.UUID
	filePath      string
	chunkSize     int64
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	workerCancels []context.CancelFunc
	onChunkSent   func(chunkIndex int64)
	onChunkFailed func(chunkIndex int64, err error)
}

// NewChunkWorkerPool creates a new worker pool
func NewChunkWorkerPool(
	workerCount int,
	queueDepth int,
	connection *quic.Conn,
	sessionKeys *crypto.SessionKeys,
	sessionID uuid.UUID,
	filePath string,
	chunkSize int64,
	onChunkSent func(chunkIndex int64),
	onChunkFailed func(chunkIndex int64, err error),
) *ChunkWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &ChunkWorkerPool{
		workerCount:   workerCount,
		chunkQueue:    make(chan int64, queueDepth),
		connection:    connection,
		sessionKeys:   sessionKeys,
		sessionID:     sessionID,
		filePath:      filePath,
		chunkSize:     chunkSize,
		ctx:           ctx,
		cancel:        cancel,
		onChunkSent:   onChunkSent,
		onChunkFailed: onChunkFailed,
		class:         PriorityP2,
	}
}

// Start starts the worker pool
func (p *ChunkWorkerPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.addWorker()
	}
}

func (p *ChunkWorkerPool) addWorker() {
	p.wg.Add(1)
	wctx, wcancel := context.WithCancel(p.ctx)
	p.workerCancels = append(p.workerCancels, wcancel)
	id := len(p.workerCancels)
	go p.workerWithCtx(id, wctx)
}

// EnqueueChunk adds a chunk to the transmission queue
func (p *ChunkWorkerPool) EnqueueChunk(chunkIndex int64) error {
	select {
	case p.chunkQueue <- chunkIndex:
		return nil
	case <-p.ctx.Done():
		return ErrWorkerPoolStopped
	}
}

// Stop stops the worker pool gracefully
func (p *ChunkWorkerPool) Stop() {
	// Stop workers
	for _, c := range p.workerCancels {
		c()
	}
	close(p.chunkQueue)
	p.wg.Wait()
	p.cancel()
}

// SetChunkSize updates the chunk size used by workers
func (p *ChunkWorkerPool) SetChunkSize(bytes int64) {
	if bytes > 0 {
		p.chunkSize = bytes
	}
}

// ScaleWorkers adjusts the number of active workers. It can scale up or down.
func (p *ChunkWorkerPool) ScaleWorkers(target int) {
	if target <= 0 {
		target = 1
	}
	// Scale up
	for len(p.workerCancels) < target {
		p.addWorker()
	}
	// Scale down
	for len(p.workerCancels) > target {
		idx := len(p.workerCancels) - 1
		p.workerCancels[idx]()
		p.workerCancels = p.workerCancels[:idx]
	}
}

// worker processes chunks from the queue
func (p *ChunkWorkerPool) workerWithCtx(workerID int, wctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case chunkIndex, ok := <-p.chunkQueue:
			if !ok {
				// Queue closed, worker exits
				return
			}

			// If a scheduler is present, enqueue by priority class
			if p.scheduler != nil {
				ci := chunkIndex
				p.scheduler.Enqueue(p.class, func(ctx context.Context) {
					if err := p.sendChunk(ci); err != nil {
						fmt.Printf("Worker %d: failed to send chunk %d: %v\n", workerID, ci, err)
						if p.onChunkFailed != nil {
							p.onChunkFailed(ci, err)
						}
						return
					}
					if p.onChunkSent != nil {
						p.onChunkSent(ci)
					}
				})
				continue
			}

			if err := p.sendChunk(chunkIndex); err != nil {
				// Log error and enqueue DTN retry via callback
				fmt.Printf("Worker %d: failed to send chunk %d: %v\n", workerID, chunkIndex, err)
				if p.onChunkFailed != nil {
					p.onChunkFailed(chunkIndex, err)
				}
				continue
			}

			// Notify chunk sent
			if p.onChunkSent != nil {
				p.onChunkSent(chunkIndex)
			}

		case <-p.ctx.Done():
			return
		case <-wctx.Done():
			return
		}
	}
}

// sendChunk sends a single chunk over a QUIC stream
func (p *ChunkWorkerPool) sendChunk(chunkIndex int64) error {
	// Open new QUIC stream for this chunk
	stream, err := p.connection.OpenStreamSync(p.ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Read chunk data from file
	chunkData, err := p.readChunkFromFile(chunkIndex)
	if err != nil {
		return err
	}

	// Encrypt chunk
	encryptedChunk, err := p.encryptChunk(chunkIndex, chunkData)
	if err != nil {
		return err
	}

	// Build chunk message header
	header := p.buildChunkHeader(chunkIndex, len(encryptedChunk))

	// Write header and encrypted payload to stream
	if _, err := stream.Write(header); err != nil {
		return err
	}

	if _, err := stream.Write(encryptedChunk); err != nil {
		return err
	}

	return nil
}

// readChunkFromFile reads chunk data from file at the appropriate offset
func (p *ChunkWorkerPool) readChunkFromFile(chunkIndex int64) ([]byte, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	offset := chunkIndex * p.chunkSize
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, err
	}

	chunkData := make([]byte, p.chunkSize)
	n, err := file.Read(chunkData)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return chunkData[:n], nil
}

// encryptChunk encrypts chunk data using session keys
func (p *ChunkWorkerPool) encryptChunk(chunkIndex int64, plaintext []byte) ([]byte, error) {
	// Derive nonce from chunk index
	nonce := crypto.DeriveNonce(p.sessionKeys.IVBase, uint64(chunkIndex))

	// Construct AAD from session ID and chunk index
	aad := make([]byte, 16+8)
	copy(aad[0:16], p.sessionID[:])
	binary.BigEndian.PutUint64(aad[16:24], uint64(chunkIndex))

	// Encrypt using AES-256-GCM
	ciphertext, err := crypto.Seal(p.sessionKeys.PayloadKey[:], nonce[:], aad, plaintext)
	if err != nil {
		return nil, err
	}

	return ciphertext, nil
}

// buildChunkHeader constructs the chunk message header
func (p *ChunkWorkerPool) buildChunkHeader(chunkIndex int64, payloadLen int) []byte {
	header := make([]byte, ChunkHeaderSize)

	// Magic (4 bytes)
	binary.BigEndian.PutUint32(header[0:4], ChunkMagic)

	// Version (1 byte)
	header[4] = ChunkVersion

	// Reserved (3 bytes) - zeros

	// SessionID (16 bytes)
	copy(header[8:24], p.sessionID[:])

	// ChunkIndex (4 bytes)
	binary.BigEndian.PutUint32(header[24:28], uint32(chunkIndex))

	// PayloadLength (4 bytes)
	binary.BigEndian.PutUint32(header[28:32], uint32(payloadLen))

	return header
}
