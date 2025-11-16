package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateEd25519 generates a new Ed25519 identity keypair.
// The keypair can be used for peer authentication and digital signatures.
//
// Returns:
//   - Ed25519KeyPair containing public and private keys
//   - error if random number generation fails
func GenerateEd25519() (*Ed25519KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
	}

	return &Ed25519KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// GenerateX25519 generates a new X25519 ephemeral keypair for key exchange.
// These keys should be generated fresh for each transfer session and destroyed
// after the session ends to ensure forward secrecy.
//
// Returns:
//   - X25519KeyPair containing public and private keys
//   - error if random number generation fails
func GenerateX25519() (*X25519KeyPair, error) {
	var kp X25519KeyPair

	// Generate random private key
	if _, err := rand.Read(kp.PrivateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate X25519 private key: %w", err)
	}

	// Derive public key from private key
	curve25519.ScalarBaseMult(&kp.PublicKey, &kp.PrivateKey)

	return &kp, nil
}

// X25519Exchange performs Elliptic Curve Diffie-Hellman key exchange.
// Given our private key and peer's public key, computes the shared secret.
//
// Parameters:
//   - ourPrivate: Our X25519 private key
//   - theirPublic: Peer's X25519 public key
//
// Returns:
//   - sharedSecret: 32-byte shared secret
//   - error if ECDH computation fails
func X25519Exchange(ourPrivate, theirPublic *[32]byte) ([32]byte, error) {
	var sharedSecret [32]byte

	// Perform scalar multiplication: sharedSecret = ourPrivate * theirPublic
	curve25519.ScalarMult(&sharedSecret, ourPrivate, theirPublic)

	// Check for all-zero output (invalid exchange)
	allZero := true
	for _, b := range sharedSecret {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		return sharedSecret, errors.New("X25519 exchange resulted in all-zero shared secret (invalid public key)")
	}

	return sharedSecret, nil
}

// SharedSecret computes the shared secret using X25519 ECDH.
// This is a convenience function that calls X25519Exchange and returns the result as []byte.
func SharedSecret(ourPrivate, theirPublic *[32]byte) []byte {
	secret, err := X25519Exchange(ourPrivate, theirPublic)
	if err != nil {
		// In case of error, return zero bytes (should not happen in normal operation)
		return make([]byte, 32)
	}
	return secret[:]
}