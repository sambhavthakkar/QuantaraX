package manager

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionAlreadyExists   = errors.New("session already exists")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)

// SessionStore manages in-memory session storage
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionStore creates a new session store
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Add adds a new session to the store
func (s *SessionStore) Add(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.sessions[session.ID]; exists {
		return ErrSessionAlreadyExists
	}
	
	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (s *SessionStore) Get(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}
	
	return session, nil
}

// Update updates an existing session
func (s *SessionStore) Update(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.sessions[session.ID]; !exists {
		return ErrSessionNotFound
	}
	
	s.sessions[session.ID] = session
	return nil
}

// Delete removes a session from the store
func (s *SessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.sessions[sessionID]; !exists {
		return ErrSessionNotFound
	}
	
	delete(s.sessions, sessionID)
	return nil
}

// List returns all sessions matching optional filter
func (s *SessionStore) List(filterState *TransferState, limit, offset int) ([]*Session, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var filtered []*Session
	for _, session := range s.sessions {
		if filterState != nil && session.State != *filterState {
			continue
		}
		filtered = append(filtered, session)
	}
	
	total := len(filtered)
	
	// Apply pagination
	if offset >= len(filtered) {
		return []*Session{}, total
	}
	
	end := offset + limit
	if end > len(filtered) || limit == 0 {
		end = len(filtered)
	}
	
	return filtered[offset:end], total
}

// CleanupOldSessions removes sessions older than the specified duration
func (s *SessionStore) CleanupOldSessions(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	
	for id, session := range s.sessions {
		// Only cleanup completed or failed sessions
		if (session.State == StateCompleted || session.State == StateFailed) &&
			session.UpdateTime.Before(cutoff) {
			delete(s.sessions, id)
			removed++
		}
	}
	
	return removed
}

// Count returns the total number of sessions
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
