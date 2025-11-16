package manager

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrDatabaseNotInitialized = errors.New("database not initialized")
	ErrBitmapNotFound         = errors.New("bitmap not found")
)

// PersistentStore manages SQLite-backed session and bitmap storage
type PersistentStore struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// NewPersistentStore creates a new persistent store with SQLite backend
func NewPersistentStore(dbPath string) (*PersistentStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	store := &PersistentStore{
		db:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initSchema creates the database schema if it doesn't exist
func (ps *PersistentStore) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS transfer_sessions (
			session_id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			chunk_size INTEGER NOT NULL,
			total_chunks INTEGER NOT NULL,
			direction TEXT NOT NULL,
			state TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			metadata TEXT
		);

		CREATE TABLE IF NOT EXISTS chunk_bitmaps (
			session_id TEXT PRIMARY KEY,
			bitmap_data BLOB NOT NULL,
			chunks_received INTEGER NOT NULL DEFAULT 0,
			last_updated TIMESTAMP NOT NULL,
			FOREIGN KEY (session_id) REFERENCES transfer_sessions(session_id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_sessions_state ON transfer_sessions(state);
		CREATE INDEX IF NOT EXISTS idx_bitmaps_updated ON chunk_bitmaps(last_updated);
	`

	if _, err := ps.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Insert schema version if not exists
	var version int
	err := ps.db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&version)
	if err == sql.ErrNoRows {
		if _, err := ps.db.Exec("INSERT INTO schema_version (version) VALUES (1)"); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query schema version: %w", err)
	}

	return nil
}

// SaveSession persists a session to the database
func (ps *PersistentStore) SaveSession(session *Session) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO transfer_sessions 
		(session_id, file_path, file_name, file_size, chunk_size, total_chunks, 
		 direction, state, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = ps.db.Exec(query,
		session.ID,
		session.FilePath,
		session.FileName,
		session.FileSize,
		session.ChunkSize,
		session.TotalChunks,
		session.Direction.String(),
		session.State.String(),
		session.StartTime,
		session.UpdateTime,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// LoadSession retrieves a session from the database
func (ps *PersistentStore) LoadSession(sessionID string) (*Session, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var (
		filePath     string
		fileName     string
		fileSize     int64
		chunkSize    int64
		totalChunks  int64
		directionStr string
		stateStr     string
		createdAt    time.Time
		updatedAt    time.Time
		metadataJSON string
	)

	query := `
		SELECT file_path, file_name, file_size, chunk_size, total_chunks,
		       direction, state, created_at, updated_at, metadata
		FROM transfer_sessions
		WHERE session_id = ?
	`

	err := ps.db.QueryRow(query, sessionID).Scan(
		&filePath, &fileName, &fileSize, &chunkSize, &totalChunks,
		&directionStr, &stateStr, &createdAt, &updatedAt, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Parse direction
	var direction TransferDirection
	switch directionStr {
	case "SEND":
		direction = DirectionSend
	case "RECEIVE":
		direction = DirectionReceive
	default:
		return nil, fmt.Errorf("invalid direction: %s", directionStr)
	}

	// Parse state
	var state TransferState
	switch stateStr {
	case "PENDING":
		state = StatePending
	case "ACTIVE":
		state = StateActive
	case "PAUSED":
		state = StatePaused
	case "COMPLETED":
		state = StateCompleted
	case "FAILED":
		state = StateFailed
	default:
		return nil, fmt.Errorf("invalid state: %s", stateStr)
	}

	session := &Session{
		ID:          sessionID,
		FilePath:    filePath,
		FileName:    fileName,
		FileSize:    fileSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		Direction:   direction,
		State:       state,
		StartTime:   createdAt,
		UpdateTime:  updatedAt,
		Metadata:    make(map[string]string),
	}

	// Parse metadata
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return session, nil
}

// UpdateSessionState updates only the session state
func (ps *PersistentStore) UpdateSessionState(sessionID string, newState TransferState) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	query := `UPDATE transfer_sessions SET state = ?, updated_at = ? WHERE session_id = ?`
	result, err := ps.db.Exec(query, newState.String(), time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session state: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession removes a session and its bitmap from the database
func (ps *PersistentStore) DeleteSession(sessionID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	tx, err := ps.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete bitmap first (foreign key constraint)
	if _, err := tx.Exec("DELETE FROM chunk_bitmaps WHERE session_id = ?", sessionID); err != nil {
		return fmt.Errorf("failed to delete bitmap: %w", err)
	}

	// Delete session
	result, err := tx.Exec("DELETE FROM transfer_sessions WHERE session_id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListSessions returns all sessions matching the filter
func (ps *PersistentStore) ListSessions(filterState *TransferState, limit, offset int) ([]*Session, int, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var sessions []*Session
	var query string
	var args []interface{}

	// Build query based on filter
	if filterState != nil {
		query = "SELECT session_id FROM transfer_sessions WHERE state = ? ORDER BY created_at DESC LIMIT ? OFFSET ?"
		args = []interface{}{filterState.String(), limit, offset}
	} else {
		query = "SELECT session_id FROM transfer_sessions ORDER BY created_at DESC LIMIT ? OFFSET ?"
		args = []interface{}{limit, offset}
	}

	rows, err := ps.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, 0, fmt.Errorf("failed to scan session ID: %w", err)
		}

		// Load full session (inefficient but simple for now)
		session, err := ps.LoadSession(sessionID)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	// Get total count
	var total int
	var countQuery string
	var countArgs []interface{}
	if filterState != nil {
		countQuery = "SELECT COUNT(*) FROM transfer_sessions WHERE state = ?"
		countArgs = []interface{}{filterState.String()}
	} else {
		countQuery = "SELECT COUNT(*) FROM transfer_sessions"
	}
	if err := ps.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return sessions, total, nil
}

// Close closes the database connection
func (ps *PersistentStore) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}
