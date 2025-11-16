// Package crypto provides cryptographic primitives for QuantaraX file transfers.
//
// This package implements:
//   - Ed25519 identity keypairs for peer authentication
//   - X25519 ephemeral keypairs for forward secrecy
//   - HKDF-based session key derivation
//   - AES-256-GCM authenticated encryption
//   - Deterministic nonce generation
//   - Secure keystore with Argon2id encryption
package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
)

// Ed25519KeyPair represents an Ed25519 identity keypair.
// The private key is 64 bytes (seed + public key concatenated).
type Ed25519KeyPair struct {
	PublicKey  ed25519.PublicKey  // 32 bytes
	PrivateKey ed25519.PrivateKey // 64 bytes
}

// X25519KeyPair represents an X25519 ephemeral keypair for key exchange.
type X25519KeyPair struct {
	PublicKey  [32]byte // 32 bytes
	PrivateKey [32]byte // 32 bytes
}

// SessionKeys contains cryptographically independent keys derived from
// the shared secret using HKDF.
type SessionKeys struct {
	PayloadKey [32]byte // AES-256 key for chunk data encryption
	ControlKey [32]byte // AES-256 key for control message encryption
	IVBase     [12]byte // Base initialization vector for nonce derivation
}

// KeystoreEntry represents an encrypted Ed25519 private key stored on disk.
type KeystoreEntry struct {
	Version       int    `json:"version"`         // Format version (currently 1)
	KDF           string `json:"kdf"`             // Key derivation function ("argon2id")
	Argon2Time    int    `json:"argon2_time"`     // Argon2 time parameter
	Argon2Memory  int    `json:"argon2_memory"`   // Argon2 memory in KiB
	Argon2Threads int    `json:"argon2_threads"`  // Argon2 parallelism
	Salt          []byte `json:"salt"`            // Random salt for KDF
	Nonce         []byte `json:"nonce"`           // Random nonce for AES-GCM
	Ciphertext    []byte `json:"ciphertext"`      // Encrypted private key + auth tag
}

// ComputeFingerprint computes a SHA-256 fingerprint of a public key
func ComputeFingerprint(publicKey ed25519.PublicKey) string {
	hash := sha256.Sum256(publicKey)
	return "SHA256:" + hex.EncodeToString(hash[:])
}
