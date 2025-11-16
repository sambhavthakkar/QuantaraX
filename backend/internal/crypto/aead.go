package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

var (
	// ErrInvalidKeySize is returned when the provided key is not 32 bytes
	ErrInvalidKeySize = errors.New("key must be exactly 32 bytes for AES-256")

	// ErrInvalidNonceSize is returned when the provided nonce is not 12 bytes
	ErrInvalidNonceSize = errors.New("nonce must be exactly 12 bytes for GCM")

	// ErrAuthenticationFailed is returned when GCM authentication tag verification fails
	ErrAuthenticationFailed = errors.New("authentication failed: ciphertext has been tampered with")
)

// Seal encrypts and authenticates plaintext using AES-256-GCM.
//
// The function:
//   1. Validates key and nonce sizes
//   2. Initializes AES-256 cipher
//   3. Creates GCM mode wrapper
//   4. Encrypts plaintext and appends 16-byte authentication tag
//
// AAD (Additional Authenticated Data) is authenticated but not encrypted.
// Use AAD for context like chunk index or session ID to prevent reordering attacks.
//
// Parameters:
//   - key: 32-byte AES-256 key (PayloadKey or ControlKey)
//   - nonce: 12-byte unique initialization vector (must be unique per encryption)
//   - aad: Additional Authenticated Data (can be nil)
//   - plaintext: Data to encrypt
//
// Returns:
//   - ciphertext: Encrypted data concatenated with 16-byte authentication tag
//   - error: Non-nil if encryption fails or invalid parameters
//
// Security Warning:
//   - NEVER reuse the same nonce with the same key
//   - Nonce reuse completely breaks GCM security
func Seal(key []byte, nonce []byte, aad []byte, plaintext []byte) ([]byte, error) {
	// Validate key size
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}

	// Validate nonce size
	if len(nonce) != 12 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidNonceSize, len(nonce))
	}

	// Initialize AES-256 cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode wrapper
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt and authenticate
	// GCM.Seal appends the ciphertext and tag to dst (nil here, so it allocates)
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)

	return ciphertext, nil
}

// Open decrypts and verifies authenticated ciphertext using AES-256-GCM.
//
// The function:
//   1. Validates key and nonce sizes
//   2. Initializes AES-256 cipher
//   3. Creates GCM mode wrapper
//   4. Decrypts and verifies authentication tag
//   5. Returns error if authentication fails (DOES NOT return partial plaintext)
//
// AAD (Additional Authenticated Data) must match the AAD used during encryption.
//
// Parameters:
//   - key: 32-byte AES-256 key (same as used for encryption)
//   - nonce: 12-byte initialization vector (same as used for encryption)
//   - aad: Additional Authenticated Data (same as used for encryption, can be nil)
//   - ciphertext: Encrypted data with appended 16-byte authentication tag
//
// Returns:
//   - plaintext: Decrypted data if authentication succeeds
//   - error: Non-nil if authentication fails or invalid parameters
//
// Security Critical:
//   - MUST verify authentication tag before returning plaintext
//   - MUST NOT return plaintext if verification fails
//   - Tag comparison is constant-time (handled by cipher.GCM)
func Open(key []byte, nonce []byte, aad []byte, ciphertext []byte) ([]byte, error) {
	// Validate key size
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}

	// Validate nonce size
	if len(nonce) != 12 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidNonceSize, len(nonce))
	}

	// Validate ciphertext minimum size (at least 16 bytes for tag)
	if len(ciphertext) < 16 {
		return nil, errors.New("ciphertext too short (must be at least 16 bytes for tag)")
	}

	// Initialize AES-256 cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode wrapper
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify authentication tag
	// GCM.Open returns error if authentication fails
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}

	return plaintext, nil
}