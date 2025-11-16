package chunker

import "time"

// FECProfile describes FEC parameters
type FECProfile struct {
	K int `json:"K"`
	R int `json:"R"`
}

// NetworkProfile captures measured network stats
type NetworkProfile struct {
	RTTMsAvg   float64 `json:"rtt_ms_avg"`
	RTTMsStd   float64 `json:"rtt_ms_std"`
	LossPct    float64 `json:"loss_pct"`
	Bandwidth  float64 `json:"bandwidth_mbps_est"`
	Reconnects int     `json:"reconnects"`
	PathChanges int    `json:"path_changes"`
}

// TransferPolicies controls ACK/resume/encryption
type TransferPolicies struct {
	AckMode string `json:"ack"`
	Resume  string `json:"resume"`
	Encryption struct {
		E2E    bool `json:"e2e"`
		AtRest bool `json:"at_rest"`
	} `json:"encryption"`
	NoRelayCache bool `json:"no_relay_cache"`
}

// MediaProfile optional media fields
type MediaProfile struct {
	Codec        string  `json:"codec,omitempty"`
	Framerate    float64 `json:"framerate,omitempty"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	MoovPosition string  `json:"moov_position,omitempty"`
	PreviewHash  string  `json:"preview_hash,omitempty"`
}

// MedicalProfile optional medical fields
type MedicalProfile struct {
	StrictMode bool `json:"strict_mode"`
	E2E        bool `json:"e2e_encryption"`
	AtRest     bool `json:"at_rest_encryption"`
	// Optional minimal DICOM metadata (redacted as needed)
	PatientID   string `json:"patient_id,omitempty"`
	StudyUID    string `json:"study_uid,omitempty"`
	SeriesCount int    `json:"series_count,omitempty"`
	Modality    string `json:"modality,omitempty"`
}

// EngineeringProfile optional engineering fields
type EngineeringProfile struct {
	Dependencies     []Dependency       `json:"dependency_graph,omitempty"`
	DeltaCheckpoints []DeltaCheckpoint  `json:"delta_checkpoints,omitempty"`
}

type DeltaCheckpoint struct {
	Path       string `json:"path"`
	BlockSize  int    `json:"block_size"`
	BlockCount int    `json:"block_count"`
}
type Dependency struct {
	Node      string   `json:"node"`
	DependsOn []string `json:"depends_on"`
}

// TelemetryProfile optional telemetry fields
type TelemetryProfile struct {
	Streams    []TelemetryStream `json:"streams,omitempty"`
	SyncPoints []SyncPoint       `json:"sync_points,omitempty"`
}

type TelemetryStream struct {
	Name        string  `json:"name"`
	Priority    int     `json:"priority"`
	SampleRate  float64 `json:"sample_rate_hz"`
	Channels    int     `json:"channels"`
}

type SyncPoint struct {
	Timestamp int64 `json:"ts"`
	VideoFrame int  `json:"video_frame"`
}

// DTNProfile optional DTN settings
type DTNProfile struct {
	TTLSeconds   int  `json:"ttl_seconds"`
	Custody      bool `json:"custody_transfer"`
	MaxRetries   int  `json:"max_retries"`
	BackoffMs    int  `json:"backoff_ms"`
}

// Manifest represents the complete file chunking metadata
type Manifest struct {
	SessionID   string            `json:"session_id"`
	FileName    string            `json:"file_name"`
	FileSize    int64             `json:"file_size"`
	ChunkSize   int               `json:"chunk_size"`
	ChunkCount  int               `json:"chunk_count"`
	HashAlgo    string            `json:"hash_algo"`
	Chunks      []ChunkDescriptor `json:"chunks"`
	MerkleRoot  string            `json:"merkle_root"`
	CreatedAt   time.Time         `json:"created_at"`

	// Domain-aware extensions (optional)
	Domain          string           `json:"domain,omitempty"`
	FEC             *FECProfile      `json:"fec_profile,omitempty"`
	Network         *NetworkProfile  `json:"network_profile,omitempty"`
	Policies        *TransferPolicies `json:"transfer_policies,omitempty"`

	// Domain-specific optional blocks
	MediaProfile       *MediaProfile       `json:"media_profile,omitempty"`
	MedicalProfile     *MedicalProfile     `json:"medical_profile,omitempty"`
	EngineeringProfile *EngineeringProfile `json:"engineering_profile,omitempty"`
	TelemetryProfile   *TelemetryProfile   `json:"telemetry_profile,omitempty"`
	DTNProfile         *DTNProfile         `json:"dtn_profile,omitempty"`
}

// ChunkDescriptor describes a single chunk
type ChunkDescriptor struct {
	Index  int    `json:"index"`
	Hash   string `json:"hash"`   // Base64-encoded BLAKE3 hash
	Length int    `json:"length"` // Actual chunk length in bytes
}

// ChunkOptions configures chunking behavior
type ChunkOptions struct {
	ChunkSize int // Chunk size in bytes (default: 1 MiB)
}

// DefaultChunkOptions returns default chunking options
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize: 1048576, // 1 MiB
	}
}