package transport

import "github.com/quantarax/backend/internal/chunker"

// ProfileForDomain returns transport class configs for a given domain.
func ProfileForDomain(domain string, manifest *chunker.Manifest) DomainTransportProfile {
	switch domain {
	case "racetrack_factory":
		// Telemetry highest priority, video headers next, bulk last
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckImmediate,  Streams: 4, ChunkBytes: 512*1024},  // telemetry
			P1: ClassConfig{Ack: AckDelayed10ms, Streams: 2, ChunkBytes: 512*1024},  // headers/keyframes
			P2: ClassConfig{Ack: AckDelayed25ms, Streams: 6, ChunkBytes: 1024*1024}, // bulk video
		}
	case "media":
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckDelayed10ms, Streams: 1,  ChunkBytes: manifest.ChunkSize}, // control-like
			P1: ClassConfig{Ack: AckDelayed10ms, Streams: 8,  ChunkBytes: 1024*1024},           // headers/keyframes
			P2: ClassConfig{Ack: AckDelayed25ms, Streams: 8,  ChunkBytes: 4*1024*1024},         // bulk video
		}
	case "engineering":
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckDelayed10ms, Streams: 1, ChunkBytes: manifest.ChunkSize},
			P1: ClassConfig{Ack: AckDelayed25ms, Streams: 4, ChunkBytes: 256*1024},
			P2: ClassConfig{Ack: AckDelayed25ms, Streams: 4, ChunkBytes: 256*1024},
		}
	case "medical":
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckImmediate,   Streams: 1, ChunkBytes: manifest.ChunkSize}, // control
			P1: ClassConfig{Ack: AckImmediate,   Streams: 2, ChunkBytes: 256*1024},
			P2: ClassConfig{Ack: AckDelayed10ms,  Streams: 2, ChunkBytes: 256*1024},
		}
	case "disaster":
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckDelayed10ms, Streams: 1, ChunkBytes: 256*1024}, // thumbs/metadata
			P1: ClassConfig{Ack: AckDelayed10ms, Streams: 1, ChunkBytes: 256*1024},
			P2: ClassConfig{Ack: AckDelayed10ms, Streams: 2, ChunkBytes: 256*1024},
		}
	case "rural":
		fallthrough
	default:
		return DomainTransportProfile{
			P0: ClassConfig{Ack: AckDelayed10ms, Streams: 1, ChunkBytes: 384*1024},
			P1: ClassConfig{Ack: AckDelayed10ms, Streams: 1, ChunkBytes: 384*1024},
			P2: ClassConfig{Ack: AckDelayed10ms, Streams: 2, ChunkBytes: 384*1024},
		}
	}
}
