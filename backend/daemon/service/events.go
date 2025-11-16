package service

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// EventType represents different event classifications
type EventType int

const (
	EventStarted EventType = iota + 1
	EventProgress
	EventPaused
	EventResumed
	EventCompleted
	EventFailed
	EventChunkSent
	EventChunkReceived
)

func (e EventType) String() string {
	switch e {
	case EventStarted:
		return "STARTED"
	case EventProgress:
		return "PROGRESS"
	case EventPaused:
		return "PAUSED"
	case EventResumed:
		return "RESUMED"
	case EventCompleted:
		return "COMPLETED"
	case EventFailed:
		return "FAILED"
	case EventChunkSent:
		return "CHUNK_SENT"
	case EventChunkReceived:
		return "CHUNK_RECEIVED"
	default:
		return "UNKNOWN"
	}
}

// TransferEvent represents a transfer-related event
type TransferEvent struct {
	SessionID       string
	EventType       EventType
	Timestamp       time.Time
	ProgressPercent float64
	Message         string
	Metadata        map[string]string
}

// EventSubscription represents an active event subscription
type EventSubscription struct {
	ID              string
	SessionIDFilter string
	Channel         chan *TransferEvent
}

// EventPublisher manages event subscriptions and broadcasting
type EventPublisher struct {
	subscriptions map[string]*EventSubscription
	mu            sync.RWMutex
	bufferSize    int
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(bufferSize int) *EventPublisher {
	return &EventPublisher{
		subscriptions: make(map[string]*EventSubscription),
		bufferSize:    bufferSize,
	}
}

// Subscribe creates a new event subscription
func (p *EventPublisher) Subscribe(sessionIDFilter string) *EventSubscription {
	p.mu.Lock()
	defer p.mu.Unlock()

	sub := &EventSubscription{
		ID:              generateSubscriptionID(),
		SessionIDFilter: sessionIDFilter,
		Channel:         make(chan *TransferEvent, p.bufferSize),
	}

	p.subscriptions[sub.ID] = sub
	return sub
}

// Unsubscribe removes an event subscription
func (p *EventPublisher) Unsubscribe(subscriptionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, exists := p.subscriptions[subscriptionID]; exists {
		close(sub.Channel)
		delete(p.subscriptions, subscriptionID)
	}
}

// Publish broadcasts an event to all matching subscribers
func (p *EventPublisher) Publish(event *TransferEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, sub := range p.subscriptions {
		// Apply session ID filter
		if sub.SessionIDFilter != "" && sub.SessionIDFilter != event.SessionID {
			continue
		}

		// Non-blocking send to prevent slow consumers from blocking
		select {
		case sub.Channel <- event:
			// Event sent successfully
		default:
			// Channel full, skip this event (slow consumer protection)
		}
	}
}

// PublishStarted publishes a transfer started event
func (p *EventPublisher) PublishStarted(sessionID, fileName string, totalSize int64) {
	p.Publish(&TransferEvent{
		SessionID:       sessionID,
		EventType:       EventStarted,
		Timestamp:       time.Now(),
		ProgressPercent: 0,
		Message:         "Transfer started",
		Metadata: map[string]string{
			"file_name":  fileName,
			"total_size": strconv.FormatInt(totalSize, 10),
		},
	})
}

// PublishProgress publishes a progress update event
func (p *EventPublisher) PublishProgress(sessionID string, progressPercent float64, transferRateMbps float64) {
	p.Publish(&TransferEvent{
		SessionID:       sessionID,
		EventType:       EventProgress,
		Timestamp:       time.Now(),
		ProgressPercent: progressPercent,
		Message:         "Transfer in progress",
		Metadata: map[string]string{
			"transfer_rate_mbps": formatFloat(transferRateMbps),
		},
	})
}

// PublishCompleted publishes a transfer completed event
func (p *EventPublisher) PublishCompleted(sessionID string, totalTime time.Duration, avgSpeed float64) {
	p.Publish(&TransferEvent{
		SessionID:       sessionID,
		EventType:       EventCompleted,
		Timestamp:       time.Now(),
		ProgressPercent: 100,
		Message:         "Transfer completed successfully",
		Metadata: map[string]string{
			"total_time_seconds": strconv.FormatInt(int64(totalTime.Seconds()), 10),
			"average_speed_mbps": formatFloat(avgSpeed),
		},
	})
}

// PublishFailed publishes a transfer failed event
func (p *EventPublisher) PublishFailed(sessionID, errorMessage string) {
	p.Publish(&TransferEvent{
		SessionID:       sessionID,
		EventType:       EventFailed,
		Timestamp:       time.Now(),
		ProgressPercent: 0,
		Message:         errorMessage,
	})
}

// PublishChunkSent publishes a chunk sent event
func (p *EventPublisher) PublishChunkSent(sessionID string, chunkIndex int64) {
	p.Publish(&TransferEvent{
		SessionID: sessionID,
		EventType: EventChunkSent,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"chunk_index": strconv.FormatInt(chunkIndex, 10),
		},
	})
}

// PublishChunkReceived publishes a chunk received event
func (p *EventPublisher) PublishChunkReceived(sessionID string, chunkIndex int64) {
	p.Publish(&TransferEvent{
		SessionID: sessionID,
		EventType: EventChunkReceived,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"chunk_index": strconv.FormatInt(chunkIndex, 10),
		},
	})
}

// GetSubscriptionCount returns the number of active subscriptions
func (p *EventPublisher) GetSubscriptionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subscriptions)
}

// Helper functions

func generateSubscriptionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
