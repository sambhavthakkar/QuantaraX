package crypto

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters (recommended values for interactive use)
	argon2Time    = 3      // Number of iterations
	argon2Memory  = 65536  // Memory in KiB (64 MiB)
	argon2Threads = 4      // Parallelism factor
	argon2KeyLen  = 32     // Output key length (AES-256)
	saltSize      = 32     // Salt size in bytes
	keystoreVersion = 1    // Keystore format version
)

var (
	// ErrInvalidPassphrase is returned when the passphrase fails to decrypt the keystore
	ErrInvalidPassphrase = errors.New("invalid passphrase or corrupted keystore")
)

// SaveKey encrypts and saves an Ed25519 private key to disk.
//
// If passphrase is empty, the key is stored unencrypted (insecure, only for testing).
// Otherwise, the key is encrypted using AES-256-GCM with a key derived from the
// passphrase using Argon2id.
//
// Parameters:
//   - privateKey: Ed25519 private key to save (64 bytes)
//   - keystorePath: Full path to the keystore file
//   - passphrase: Passphrase for encryption (empty = no encryption)
//
// Returns:
//   - error if saving fails
func SaveKey(privateKey []byte, keystorePath string, passphrase string) error {
	if len(privateKey) != 64 {
		return errors.New("Ed25519 private key must be 64 bytes")
	}

	// Ensure directory exists
	dir := filepath.Dir(keystorePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	var data []byte

	if passphrase == "" {
		// Store unencrypted (insecure, for testing only)
		data = privateKey
		keystorePath += ".insecure"
	} else {
		// Encrypt with Argon2id + AES-256-GCM
		entry, err := encryptKey(privateKey, passphrase)
		if err != nil {
			return fmt.Errorf("failed to encrypt key: %w", err)
		}

		var marshalErr error
		data, marshalErr = json.MarshalIndent(entry, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal keystore entry: %w", marshalErr)
		}
	}

	// Write to file with restricted permissions (owner read/write only)
	if err := os.WriteFile(keystorePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write keystore file: %w", err)
	}

	return nil
}

// LoadKey loads and decrypts an Ed25519 private key from disk.
//
// If the keystore file ends with ".insecure", it is loaded without decryption.
// Otherwise, the passphrase is used to decrypt the key.
//
// Parameters:
//   - keystorePath: Full path to the keystore file
//   - passphrase: Passphrase for decryption (ignored for .insecure files)
//
// Returns:
//   - privateKey: Ed25519 private key (64 bytes)
//   - error if loading or decryption fails
func LoadKey(keystorePath string, passphrase string) ([]byte, error) {
	data, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Check if unencrypted
	if filepath.Ext(keystorePath) == ".insecure" {
		if len(data) != 64 {
			return nil, errors.New("invalid unencrypted keystore: expected 64 bytes")
		}
		return data, nil
	}

	// Decrypt encrypted keystore
	var entry KeystoreEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keystore entry: %w", err)
	}

	privateKey, err := decryptKey(&entry, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return privateKey, nil
}

// encryptKey encrypts an Ed25519 private key using Argon2id + AES-256-GCM.
func encryptKey(privateKey []byte, passphrase string) (*KeystoreEntry, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key from passphrase using Argon2id
	derivedKey := argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	// Generate random nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt private key using AES-256-GCM (no AAD for keystore)
	ciphertext, err := Seal(derivedKey, nonce, nil, privateKey)
	if err != nil {
		return nil, err
	}

	entry := &KeystoreEntry{
		Version:       keystoreVersion,
		KDF:           "argon2id",
		Argon2Time:    argon2Time,
		Argon2Memory:  argon2Memory,
		Argon2Threads: argon2Threads,
		Salt:          salt,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}

	return entry, nil
}

// decryptKey decrypts an Ed25519 private key using Argon2id + AES-256-GCM.
func decryptKey(entry *KeystoreEntry, passphrase string) ([]byte, error) {
	// Validate keystore version
	if entry.Version != keystoreVersion {
		return nil, fmt.Errorf("unsupported keystore version: %d", entry.Version)
	}

	// Validate KDF
	if entry.KDF != "argon2id" {
		return nil, fmt.Errorf("unsupported KDF: %s", entry.KDF)
	}

	// Derive decryption key from passphrase using stored parameters
	derivedKey := argon2.IDKey(
		[]byte(passphrase),
		entry.Salt,
		uint32(entry.Argon2Time),
		uint32(entry.Argon2Memory),
		uint8(entry.Argon2Threads),
		argon2KeyLen,
	)

	// Decrypt private key using AES-256-GCM
	plaintext, err := Open(derivedKey, entry.Nonce, nil, entry.Ciphertext)
	if err != nil {
		return nil, ErrInvalidPassphrase
	}

	// Validate decrypted key size
	if len(plaintext) != 64 {
		return nil, errors.New("decrypted key has invalid size")
	}

	return plaintext, nil
}

// GetDefaultKeystorePath returns the default keystore directory path.
// On Windows: %APPDATA%\quantarax\keys
// On Unix: $XDG_DATA_HOME/quantarax/keys or ~/.local/share/quantarax/keys
func GetDefaultKeystorePath() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		// Windows
		return filepath.Join(appData, "quantarax", "keys")
	}

	// Unix-like
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "quantarax", "keys")
	}

	// Fallback to ~/.local/share
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share", "quantarax", "keys")
}