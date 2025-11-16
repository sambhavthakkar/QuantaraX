package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// Domain separation string for session key derivation
	sessionInfoString = "quantarax-v1-session"

	// Expected output length from HKDF: 32 (PayloadKey) + 32 (ControlKey) + 12 (IVBase) = 76 bytes
	hkdfOutputLength = 76
)

// DeriveSessionKeys performs HKDF-based key derivation from an X25519 shared secret.
//
// This function derives three cryptographically independent keys:
//   - PayloadKey: For encrypting file chunk data (AES-256-GCM)
//   - ControlKey: For encrypting control messages (AES-256-GCM)
//   - IVBase: For deterministic nonce generation
//
// The manifest hash is used as the HKDF salt to bind the session keys to a specific
// file transfer, ensuring keys cannot be reused across different files.
//
// Parameters:
//   - ourPrivate: Our X25519 private key
//   - theirPublic: Peer's X25519 public key
//   - manifestHash: BLAKE3 hash of the file manifest (32 bytes, used as salt)
//
// Returns:
//   - SessionKeys containing PayloadKey, ControlKey, and IVBase
//   - error if ECDH fails or key derivation fails
func DeriveSessionKeys(ourPrivate, theirPublic *[32]byte, manifestHash []byte) (*SessionKeys, error) {
	// Validate manifest hash length
	if len(manifestHash) != 32 {
		return nil, fmt.Errorf("manifest hash must be 32 bytes, got %d", len(manifestHash))
	}

	// Step 1: Perform X25519 ECDH to get shared secret
	sharedSecret, err := X25519Exchange(ourPrivate, theirPublic)
	if err != nil {
		return nil, fmt.Errorf("ECDH key exchange failed: %w", err)
	}

	// Step 2: Use HKDF to derive session keys
	// - IKM (Input Key Material): shared secret from ECDH
	// - Salt: manifest hash (binds keys to specific file)
	// - Info: domain separation string
	// - Output: 76 bytes (32 + 32 + 12)
	hkdfReader := hkdf.New(
		sha256.New,
		sharedSecret[:],     // IKM
		manifestHash,        // Salt
		[]byte(sessionInfoString), // Info
	)

	// Step 3: Read derived key material
	keyMaterial := make([]byte, hkdfOutputLength)
	if _, err := io.ReadFull(hkdfReader, keyMaterial); err != nil {
		return nil, fmt.Errorf("HKDF key derivation failed: %w", err)
	}

	// Step 4: Split key material into separate keys
	var keys SessionKeys
	copy(keys.PayloadKey[:], keyMaterial[0:32])
	copy(keys.ControlKey[:], keyMaterial[32:64])
	copy(keys.IVBase[:], keyMaterial[64:76])

	return &keys, nil
}