package transport

// PriorityClass defines stream/task priority classes
// P0: highest (e.g., telemetry), P1: headers/keyframes, P2: bulk

type PriorityClass uint8

const (
	PriorityP0 PriorityClass = iota
	PriorityP1
	PriorityP2
)

// AckStrategy defines ACK timing/behavior hints
// These are hints for scheduling and pacing; QUIC-go itself manages ACKs.
// We keep these for future extensibility and observability.

type AckStrategy string

const (
	AckImmediate    AckStrategy = "immediate"
	AckDelayed10ms  AckStrategy = "delayed-10ms"
	AckDelayed25ms  AckStrategy = "delayed-25ms"
	AckMixed        AckStrategy = "mixed" // class-based
)

// ClassConfig describes per-class behavior

type ClassConfig struct {
	Ack        AckStrategy
	Streams    int // target parallel streams for this class
	ChunkBytes int // preferred chunk size
}

// DomainTransportProfile captures per-domain class configs

type DomainTransportProfile struct {
	P0, P1, P2 ClassConfig
}
