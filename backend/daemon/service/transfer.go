package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/quantarax/backend/daemon/manager"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/engineering"
	"github.com/quantarax/backend/internal/introspect"
	"github.com/quantarax/backend/internal/media"
	"strings"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrInvalidToken    = errors.New("invalid transfer token")
)

// TransferService manages file transfer operations
type TransferService struct {
	store          *manager.SessionStore
	eventPublisher *EventPublisher
	keysDir        string
	chunkSize      int64
	privateKey     ed25519.PrivateKey
	publicKey      ed25519.PublicKey
}

// NewTransferService creates a new transfer service
func NewTransferService(
	store *manager.SessionStore,
	eventPublisher *EventPublisher,
	keysDir string,
	chunkSize int64,
) (*TransferService, error) {
	// Load identity keys
	privateKey, publicKey, err := loadIdentityKeys(keysDir)
	if err != nil {
		return nil, err
	}

	ts := &TransferService{
		store:          store,
		eventPublisher: eventPublisher,
		keysDir:        keysDir,
		chunkSize:      chunkSize,
		privateKey:     privateKey,
		publicKey:      publicKey,
	}
	// Initialize DTN queue/worker
	_ = InitDTN(filepath.Join(keysDir, "dtn_queue.db"))
	return ts, nil
}

// CreateTransfer initiates a new file transfer
func (s *TransferService) CreateTransfer(
	filePath string,
	recipientID string,
	chunkSizeOverride int64,
	metadata map[string]string,
) (sessionID string, token string, manifest *chunker.Manifest, err error) {
	// Validate file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", "", nil, err
	}

	// Use override chunk size if provided
	chunkSize := s.chunkSize
	if chunkSizeOverride > 0 {
		chunkSize = chunkSizeOverride
	}

	// Auto domain detection and config mapping
	decision := introspect.Decide(filePath)
	chunkSizeToUse := int(chunkSize)
	switch decision.Domain {
	case introspect.DomainMedical:
		if chunkSizeToUse > 512*1024 {
			chunkSizeToUse = 512 * 1024
		}
	case introspect.DomainMedia:
		if chunkSizeToUse < 2*1024*1024 {
			chunkSizeToUse = 2 * 1024 * 1024
		}
	case introspect.DomainEngineering:
		if chunkSizeToUse > 256*1024 {
			chunkSizeToUse = 256 * 1024
		}
	case introspect.DomainTelemetry:
		if chunkSizeToUse < 512*1024 {
			chunkSizeToUse = 512 * 1024
		}
	case introspect.DomainDisaster, introspect.DomainRural:
		if chunkSizeToUse > 512*1024 {
			chunkSizeToUse = 512 * 1024
		}
	}

	// Generate manifest
	manifest, err = chunker.ComputeManifest(filePath, chunker.ChunkOptions{ChunkSize: chunkSizeToUse})
	if err != nil {
		return "", "", nil, err
	}
	// Domain-specific pre-processing (placeholders kept safe by default)
	if decision.Domain == introspect.DomainMedia {
		// Try moov relocation (non-destructive placeholder)
		// newPath, _ := media.RelocateMoovToFront(filePath) // disabled until full rewrite support
		_ = filePath
	}
	// Enrich manifest with domain decision and minimal policy/FEC defaults
	manifest.Domain = decision.Domain
	// Populate a coarse network profile based on domain defaults (will be refined runtime)
	np := &chunker.NetworkProfile{}
	switch decision.Domain {
	case introspect.DomainMedical:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 120, 0.5, 20
		manifest.FEC = &chunker.FECProfile{K: 16, R: 8}
		manifest.Policies = &chunker.TransferPolicies{}
		manifest.Policies.Encryption.E2E = true
		manifest.Policies.Encryption.AtRest = true
		manifest.Policies.NoRelayCache = true
		manifest.MedicalProfile = &chunker.MedicalProfile{StrictMode: true, E2E: true, AtRest: true}
	case introspect.DomainMedia:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 60, 1.0, 300
		manifest.FEC = &chunker.FECProfile{K: 50, R: 6}
		manifest.MediaProfile = &chunker.MediaProfile{}
	case introspect.DomainEngineering:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 80, 2.0, 80
		manifest.FEC = &chunker.FECProfile{K: 32, R: 4}
		manifest.EngineeringProfile = &chunker.EngineeringProfile{}
	case introspect.DomainTelemetry:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 40, 1.0, 150
		manifest.FEC = &chunker.FECProfile{K: 50, R: 8}
		manifest.TelemetryProfile = &chunker.TelemetryProfile{Streams: []chunker.TelemetryStream{{Name: "telemetry", Priority: 0, SampleRate: 1000, Channels: 8}}}
	case introspect.DomainDisaster:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 600, 20.0, 2
		manifest.FEC = &chunker.FECProfile{K: 20, R: 12}
		manifest.DTNProfile = &chunker.DTNProfile{TTLSeconds: 86400, Custody: true, MaxRetries: 10, BackoffMs: 60000}
	case introspect.DomainRural:
		np.RTTMsAvg, np.LossPct, np.Bandwidth = 300, 10.0, 5
		manifest.FEC = &chunker.FECProfile{K: 20, R: 10}
		manifest.DTNProfile = &chunker.DTNProfile{TTLSeconds: 43200, Custody: true, MaxRetries: 8, BackoffMs: 30000}
	}
	manifest.Network = np

	// Generate session ID
	sessionID = uuid.New().String()

	// Create session
	session := manager.NewSession(
		sessionID,
		filePath,
		filepath.Base(filePath),
		fileInfo.Size(),
		int64(manifest.ChunkSize),
		manager.DirectionSend,
	)
	session.Metadata = metadata

	// Add to store
	if err := s.store.Add(session); err != nil {
		return "", "", nil, err
	}

	// Generate transfer token
	token, err = s.generateToken(sessionID, manifest)
	if err != nil {
		return "", "", nil, err
	}

	// Domain-specific manifest enrichment (previews, deps, medical metadata)
	switch decision.Domain {
	case introspect.DomainMedia:
		// Attempt to generate a thumbnail alongside the file
		thumbPath := filepath.Join(filepath.Dir(filePath), filepath.Base(filePath)+".preview.jpg")
		if err := media.GenerateThumbnail(filePath, thumbPath, 512, 512); err == nil {
			// compute hash
			hash := crypto.ComputeFileHashB64(thumbPath)
			if manifest.MediaProfile == nil {
				manifest.MediaProfile = &chunker.MediaProfile{}
			}
			manifest.MediaProfile.PreviewHash = hash
			// Detect moov position if mp4/mov
			if ext := strings.ToLower(filepath.Ext(filePath)); ext == ".mp4" || ext == ".mov" {
				manifest.MediaProfile.MoovPosition = media.DetectMoovPosition(filePath)
			}
		}
	case introspect.DomainEngineering:
		deps, _ := engineering.DiscoverDependencies(filepath.Dir(filePath))
		if manifest.EngineeringProfile == nil {
			manifest.EngineeringProfile = &chunker.EngineeringProfile{}
		}
		for _, d := range deps {
			manifest.EngineeringProfile.Dependencies = append(manifest.EngineeringProfile.Dependencies, chunker.Dependency{Node: filePath, DependsOn: []string{d}})
		}
		// Compute simple block map and record a delta checkpoint for sender planning
		if blocks, err := engineering.ComputeDeltaBlocks(filePath, manifest.ChunkSize); err == nil {
			manifest.EngineeringProfile.DeltaCheckpoints = append(manifest.EngineeringProfile.DeltaCheckpoints, chunker.DeltaCheckpoint{
				Path:       filePath,
				BlockSize:  manifest.ChunkSize,
				BlockCount: len(blocks),
			})
		}
	case introspect.DomainMedical:
		if manifest.MedicalProfile == nil {
			manifest.MedicalProfile = &chunker.MedicalProfile{}
		}
		manifest.MedicalProfile.StrictMode = true
		manifest.MedicalProfile.E2E = true
		manifest.MedicalProfile.AtRest = true
		// Fill minimal metadata if available in future extractor (kept empty for now)
	}

	// Publish started event
	s.eventPublisher.PublishStarted(sessionID, filepath.Base(filePath), fileInfo.Size())

	return sessionID, token, manifest, nil
}

// AcceptTransfer accepts an incoming transfer
func (s *TransferService) AcceptTransfer(
	token string,
	outputPath string,
	resumeSessionID string,
) (sessionID string, manifest *chunker.Manifest, err error) {
	// Parse token
	sessionID, manifest, err = s.parseToken(token)
	if err != nil {
		return "", nil, err
	}

	// Enforce medical strict gating
	if manifest.Domain == "medical" {
		if manifest.Policies == nil || !manifest.Policies.Encryption.E2E || !manifest.Policies.Encryption.AtRest {
			return "", nil, errors.New("medical strict mode: encryption required")
		}
	}

	// Create session
	session := manager.NewSession(
		sessionID,
		outputPath,
		filepath.Base(outputPath),
		manifest.FileSize,
		int64(manifest.ChunkSize),
		manager.DirectionReceive,
	)

	// Add to store
	if err := s.store.Add(session); err != nil {
		return "", nil, err
	}

	return sessionID, manifest, nil
}

// GetTransferStatus retrieves transfer status
func (s *TransferService) GetTransferStatus(sessionID string) (*TransferStatus, error) {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	status := &TransferStatus{
		State:                  session.State,
		ProgressPercent:        session.GetProgressPercent(),
		ChunksTransferred:      session.ChunksTransferred,
		TotalChunks:            session.TotalChunks,
		BytesTransferred:       session.BytesTransferred,
		TransferRateMbps:       session.GetTransferRate(),
		EstimatedTimeRemaining: session.GetEstimatedTimeRemaining(),
		ErrorMessage:           session.ErrorMessage,
	}

	return status, nil
}

// ListTransfers lists active transfers
func (s *TransferService) ListTransfers(filterState *manager.TransferState, limit, offset int) ([]*manager.Session, int) {
	return s.store.List(filterState, limit, offset)
}

// GetPublicKey returns the daemon's public key
func (s *TransferService) GetPublicKey() (string, string) {
	pubKeyB64 := base64.StdEncoding.EncodeToString(s.publicKey)
	fingerprint := crypto.ComputeFingerprint(s.publicKey)
	return pubKeyB64, fingerprint
}

// generateToken creates a transfer token
func (s *TransferService) generateToken(sessionID string, manifest *chunker.Manifest) (string, error) {
	tokenData := map[string]interface{}{
		"session_id": sessionID,
		"manifest":   manifest,
		"created_at": time.Now().Unix(),
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(data)
	return "quantarax://xfer?t=" + token, nil
}

// parseToken parses a transfer token
func (s *TransferService) parseToken(token string) (string, *chunker.Manifest, error) {
	// Remove protocol prefix
	const prefix = "quantarax://xfer?t="
	if len(token) < len(prefix) {
		return "", nil, ErrInvalidToken
	}

	encoded := token[len(prefix):]
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, ErrInvalidToken
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return "", nil, ErrInvalidToken
	}

	sessionID := tokenData["session_id"].(string)

	// Parse manifest
	manifestData, err := json.Marshal(tokenData["manifest"])
	if err != nil {
		return "", nil, err
	}

	var manifest chunker.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return "", nil, err
	}

	return sessionID, &manifest, nil
}

// TransferStatus represents transfer status information
type TransferStatus struct {
	State                  manager.TransferState
	ProgressPercent        float64
	ChunksTransferred      int64
	TotalChunks            int64
	BytesTransferred       int64
	TransferRateMbps       float64
	EstimatedTimeRemaining int64
	ErrorMessage           string
}

// loadIdentityKeys loads Ed25519 keys from keystore
func loadIdentityKeys(keysDir string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	// For simplicity, generate new keys if not found
	// In production, this would load from encrypted keystore
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	return privKey, pubKey, nil
}
