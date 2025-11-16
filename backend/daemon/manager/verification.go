package manager

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"
)

// VerificationStatus represents the result of Merkle root verification
type VerificationStatus int

const (
	VerificationSuccess VerificationStatus = iota + 1
	VerificationHashMismatch
	VerificationCorruptionDetected
)

func (vs VerificationStatus) String() string {
	switch vs {
	case VerificationSuccess:
		return "SUCCESS"
	case VerificationHashMismatch:
		return "HASH_MISMATCH"
	case VerificationCorruptionDetected:
		return "CORRUPTION_DETECTED"
	default:
		return "UNKNOWN"
	}
}

// VerificationResult represents the outcome of file integrity verification
type VerificationResult struct {
	SessionID          string
	Status             VerificationStatus
	MerkleRootComputed []byte
	MerkleRootExpected []byte
	Timestamp          time.Time
	Signature          []byte
	PublicKey          []byte
}

// MerkleVerifier handles Merkle root verification for transfers
type MerkleVerifier struct{}

// NewMerkleVerifier creates a new Merkle verifier
func NewMerkleVerifier() *MerkleVerifier {
	return &MerkleVerifier{}
}

// VerifyMerkleRoot verifies that computed Merkle root matches expected
func (mv *MerkleVerifier) VerifyMerkleRoot(computed, expected []byte) VerificationStatus {
	if len(computed) != len(expected) {
		return VerificationCorruptionDetected
	}

	// Compare byte-by-byte
	for i := range computed {
		if computed[i] != expected[i] {
			return VerificationHashMismatch
		}
	}

	return VerificationSuccess
}

// SignVerificationResult signs the verification result with Ed25519
func (mv *MerkleVerifier) SignVerificationResult(
	result *VerificationResult,
	privateKey ed25519.PrivateKey,
) error {
	// Create canonical JSON for signing
	canonical, err := json.Marshal(map[string]interface{}{
		"session_id":           result.SessionID,
		"status":               result.Status.String(),
		"merkle_root_computed": result.MerkleRootComputed,
		"merkle_root_expected": result.MerkleRootExpected,
		"timestamp":            result.Timestamp.Unix(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal verification result: %w", err)
	}

	// Sign the canonical JSON
	signature := ed25519.Sign(privateKey, canonical)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	result.Signature = signature
	result.PublicKey = publicKey

	return nil
}

// VerifySignature verifies the signature on a verification result
func (mv *MerkleVerifier) VerifySignature(result *VerificationResult) bool {
	// Recreate canonical JSON
	canonical, err := json.Marshal(map[string]interface{}{
		"session_id":           result.SessionID,
		"status":               result.Status.String(),
		"merkle_root_computed": result.MerkleRootComputed,
		"merkle_root_expected": result.MerkleRootExpected,
		"timestamp":            result.Timestamp.Unix(),
	})
	if err != nil {
		return false
	}

	// Verify signature
	return ed25519.Verify(result.PublicKey, canonical, result.Signature)
}

// CreateVerificationResult creates a new verification result
func (mv *MerkleVerifier) CreateVerificationResult(
	sessionID string,
	computed, expected []byte,
) *VerificationResult {
	status := mv.VerifyMerkleRoot(computed, expected)

	return &VerificationResult{
		SessionID:          sessionID,
		Status:             status,
		MerkleRootComputed: computed,
		MerkleRootExpected: expected,
		Timestamp:          time.Now(),
	}
}
