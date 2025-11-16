package manager

import (
	"sync"
	"time"
)

// TransferState represents the state of a transfer session
type TransferState int

const (
	StatePending TransferState = iota + 1
	StateActive
	StatePaused
	StateCompleted
	StateFailed
)

func (s TransferState) String() string {
	switch s {
	case StatePending:
		return "PENDING"
	case StateActive:
		return "ACTIVE"
	case StatePaused:
		return "PAUSED"
	case StateCompleted:
		return "COMPLETED"
	case StateFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// TransferDirection indicates send or receive
type TransferDirection int

const (
	DirectionSend TransferDirection = iota + 1
	DirectionReceive
)

func (d TransferDirection) String() string {
	switch d {
	case DirectionSend:
		return "SEND"
	case DirectionReceive:
		return "RECEIVE"
	default:
		return "UNKNOWN"
	}
}

// Session represents a file transfer session
type Session struct {
	ID                string
	FilePath          string
	FileName          string
	FileSize          int64
	ChunkSize         int64
	TotalChunks       int64
	State             TransferState
	Direction         TransferDirection
	BytesTransferred  int64
	ChunksTransferred int64
	StartTime         time.Time
	UpdateTime        time.Time
	ErrorMessage      string
	Metadata          map[string]string

	// Transfer metrics
	transferRateSamples  []float64
	lastUpdateTime       time.Time
	lastBytesTransferred int64

	mu sync.RWMutex
}

// NewSession creates a new transfer session
func NewSession(id, filePath, fileName string, fileSize, chunkSize int64, direction TransferDirection) *Session {
	totalChunks := fileSize / chunkSize
	if fileSize%chunkSize != 0 {
		totalChunks++
	}

	return &Session{
		ID:                  id,
		FilePath:            filePath,
		FileName:            fileName,
		FileSize:            fileSize,
		ChunkSize:           chunkSize,
		TotalChunks:         totalChunks,
		State:               StatePending,
		Direction:           direction,
		StartTime:           time.Now(),
		UpdateTime:          time.Now(),
		Metadata:            make(map[string]string),
		transferRateSamples: make([]float64, 0, 10),
		lastUpdateTime:      time.Now(),
	}
}

// UpdateProgress updates session progress metrics
func (s *Session) UpdateProgress(bytesTransferred, chunksTransferred int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	duration := now.Sub(s.lastUpdateTime).Seconds()

	if duration > 0 {
		bytesDelta := bytesTransferred - s.lastBytesTransferred
		rate := float64(bytesDelta) / duration / 1024 / 1024 * 8 // Mbps

		s.transferRateSamples = append(s.transferRateSamples, rate)
		if len(s.transferRateSamples) > 10 {
			s.transferRateSamples = s.transferRateSamples[1:]
		}
	}

	s.BytesTransferred = bytesTransferred
	s.ChunksTransferred = chunksTransferred
	s.UpdateTime = now
	s.lastUpdateTime = now
	s.lastBytesTransferred = bytesTransferred
}

// GetTransferRate returns the current transfer rate in Mbps
func (s *Session) GetTransferRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.transferRateSamples) == 0 {
		return 0
	}

	var sum float64
	for _, rate := range s.transferRateSamples {
		sum += rate
	}
	return sum / float64(len(s.transferRateSamples))
}

// GetProgressPercent returns completion percentage
func (s *Session) GetProgressPercent() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalChunks == 0 {
		return 0
	}
	return float64(s.ChunksTransferred) / float64(s.TotalChunks) * 100
}

// GetEstimatedTimeRemaining returns estimated seconds until completion
func (s *Session) GetEstimatedTimeRemaining() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rate := s.GetTransferRate()
	if rate == 0 {
		return 0
	}

	remainingBytes := s.FileSize - s.BytesTransferred
	remainingSeconds := float64(remainingBytes) / (rate * 1024 * 1024 / 8)
	return int64(remainingSeconds)
}

// TransitionTo transitions the session to a new state
func (s *Session) TransitionTo(newState TransferState, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate state transition
	validTransitions := map[TransferState][]TransferState{
		StatePending:   {StateActive, StateFailed},
		StateActive:    {StatePaused, StateCompleted, StateFailed},
		StatePaused:    {StateActive, StateFailed},
		StateCompleted: {},
		StateFailed:    {},
	}

	allowed := validTransitions[s.State]
	isValid := false
	for _, allowedState := range allowed {
		if allowedState == newState {
			isValid = true
			break
		}
	}

	if !isValid {
		return ErrInvalidStateTransition
	}

	s.State = newState
	s.UpdateTime = time.Now()
	if errorMsg != "" {
		s.ErrorMessage = errorMsg
	}

	return nil
}

// GetState returns current state (thread-safe)
func (s *Session) GetState() TransferState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}
