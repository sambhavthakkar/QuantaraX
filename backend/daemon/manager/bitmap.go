package manager

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ChunkBitmap tracks which chunks have been received/sent
type ChunkBitmap struct {
	sessionID      string
	totalChunks    int64
	bitmap         []byte
	chunksReceived int64
	mu             sync.RWMutex
}

// NewChunkBitmap creates a new chunk bitmap for a session
func NewChunkBitmap(sessionID string, totalChunks int64) *ChunkBitmap {
	// Calculate bytes needed (1 bit per chunk)
	bitmapSize := (totalChunks + 7) / 8

	return &ChunkBitmap{
		sessionID:      sessionID,
		totalChunks:    totalChunks,
		bitmap:         make([]byte, bitmapSize),
		chunksReceived: 0,
	}
}

// SetChunk marks a chunk as received
func (cb *ChunkBitmap) SetChunk(chunkIndex int64) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if chunkIndex < 0 || chunkIndex >= cb.totalChunks {
		return fmt.Errorf("chunk index out of range: %d", chunkIndex)
	}

	byteIndex := chunkIndex / 8
	bitIndex := chunkIndex % 8

	// Check if already set
	if cb.bitmap[byteIndex]&(1<<bitIndex) != 0 {
		return nil // Already set
	}

	// Set the bit
	cb.bitmap[byteIndex] |= (1 << bitIndex)
	cb.chunksReceived++

	return nil
}

// HasChunk checks if a chunk has been received
func (cb *ChunkBitmap) HasChunk(chunkIndex int64) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if chunkIndex < 0 || chunkIndex >= cb.totalChunks {
		return false
	}

	byteIndex := chunkIndex / 8
	bitIndex := chunkIndex % 8

	return cb.bitmap[byteIndex]&(1<<bitIndex) != 0
}

// GetMissing returns indices of all missing chunks
func (cb *ChunkBitmap) GetMissing() []int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var missing []int64
	for i := int64(0); i < cb.totalChunks; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if cb.bitmap[byteIndex]&(1<<bitIndex) == 0 {
			missing = append(missing, i)
		}
	}

	return missing
}

// GetReceived returns indices of all received chunks
func (cb *ChunkBitmap) GetReceived() []int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var received []int64
	for i := int64(0); i < cb.totalChunks; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if cb.bitmap[byteIndex]&(1<<bitIndex) != 0 {
			received = append(received, i)
		}
	}

	return received
}

// GetProgress returns the number of chunks received
func (cb *ChunkBitmap) GetProgress() (received, total int64) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.chunksReceived, cb.totalChunks
}

// IsComplete checks if all chunks have been received
func (cb *ChunkBitmap) IsComplete() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.chunksReceived == cb.totalChunks
}

// Clear resets the bitmap
func (cb *ChunkBitmap) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for i := range cb.bitmap {
		cb.bitmap[i] = 0
	}
	cb.chunksReceived = 0
}

// Serialize returns the bitmap data for persistence
func (cb *ChunkBitmap) Serialize() []byte {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Make a copy to avoid race conditions
	data := make([]byte, len(cb.bitmap))
	copy(data, cb.bitmap)
	return data
}

// Deserialize loads bitmap data from persistence
func (cb *ChunkBitmap) Deserialize(data []byte) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if len(data) != len(cb.bitmap) {
		return fmt.Errorf("bitmap size mismatch: expected %d, got %d", len(cb.bitmap), len(data))
	}

	copy(cb.bitmap, data)

	// Recalculate chunks received
	cb.chunksReceived = 0
	for i := int64(0); i < cb.totalChunks; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if cb.bitmap[byteIndex]&(1<<bitIndex) != 0 {
			cb.chunksReceived++
		}
	}

	return nil
}

// BitmapStore manages persistent chunk bitmaps
type BitmapStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewBitmapStore creates a new bitmap store
func NewBitmapStore(db *sql.DB) *BitmapStore {
	return &BitmapStore{
		db: db,
	}
}

// SaveBitmap persists a chunk bitmap to the database
func (bs *BitmapStore) SaveBitmap(bitmap *ChunkBitmap) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	query := `
		INSERT OR REPLACE INTO chunk_bitmaps 
		(session_id, bitmap_data, chunks_received, last_updated)
		VALUES (?, ?, ?, ?)
	`

	_, err := bs.db.Exec(query,
		bitmap.sessionID,
		bitmap.Serialize(),
		bitmap.chunksReceived,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save bitmap: %w", err)
	}

	return nil
}

// LoadBitmap retrieves a chunk bitmap from the database
func (bs *BitmapStore) LoadBitmap(sessionID string, totalChunks int64) (*ChunkBitmap, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	var (
		bitmapData     []byte
		chunksReceived int64
		lastUpdated    time.Time
	)

	query := `
		SELECT bitmap_data, chunks_received, last_updated
		FROM chunk_bitmaps
		WHERE session_id = ?
	`

	err := bs.db.QueryRow(query, sessionID).Scan(&bitmapData, &chunksReceived, &lastUpdated)
	if err == sql.ErrNoRows {
		return nil, ErrBitmapNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to load bitmap: %w", err)
	}

	bitmap := NewChunkBitmap(sessionID, totalChunks)
	if err := bitmap.Deserialize(bitmapData); err != nil {
		return nil, fmt.Errorf("failed to deserialize bitmap: %w", err)
	}

	return bitmap, nil
}

// SetChunkPersistent marks a chunk as received and persists to database
func (bs *BitmapStore) SetChunkPersistent(bitmap *ChunkBitmap, chunkIndex int64) error {
	// Set the chunk in memory
	if err := bitmap.SetChunk(chunkIndex); err != nil {
		return err
	}

	// Persist to database
	return bs.SaveBitmap(bitmap)
}

// DeleteBitmap removes a bitmap from the database
func (bs *BitmapStore) DeleteBitmap(sessionID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	result, err := bs.db.Exec("DELETE FROM chunk_bitmaps WHERE session_id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete bitmap: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBitmapNotFound
	}

	return nil
}
