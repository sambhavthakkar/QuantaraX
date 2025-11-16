package crypto

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateEd25519 tests Ed25519 keypair generation
func TestGenerateEd25519(t *testing.T) {
	kp, err := GenerateEd25519()
	if err != nil {
		t.Fatalf("GenerateEd25519() failed: %v", err)
	}

	if len(kp.PublicKey) != 32 {
		t.Errorf("Public key length = %d, want 32", len(kp.PublicKey))
	}

	if len(kp.PrivateKey) != 64 {
		t.Errorf("Private key length = %d, want 64", len(kp.PrivateKey))
	}
}

// TestGenerateX25519 tests X25519 keypair generation
func TestGenerateX25519(t *testing.T) {
	kp, err := GenerateX25519()
	if err != nil {
		t.Fatalf("GenerateX25519() failed: %v", err)
	}

	// Check that public and private keys are not all zeros
	var zeroKey [32]byte
	if bytes.Equal(kp.PublicKey[:], zeroKey[:]) {
		t.Error("Public key is all zeros")
	}

	if bytes.Equal(kp.PrivateKey[:], zeroKey[:]) {
		t.Error("Private key is all zeros")
	}
}

// TestX25519Exchange tests ECDH key exchange produces identical shared secrets
func TestX25519Exchange(t *testing.T) {
	// Alice generates keypair
	alice, err := GenerateX25519()
	if err != nil {
		t.Fatalf("Failed to generate Alice's keypair: %v", err)
	}

	// Bob generates keypair
	bob, err := GenerateX25519()
	if err != nil {
		t.Fatalf("Failed to generate Bob's keypair: %v", err)
	}

	// Alice computes shared secret using her private key and Bob's public key
	aliceShared, err := X25519Exchange(&alice.PrivateKey, &bob.PublicKey)
	if err != nil {
		t.Fatalf("Alice's X25519Exchange failed: %v", err)
	}

	// Bob computes shared secret using his private key and Alice's public key
	bobShared, err := X25519Exchange(&bob.PrivateKey, &alice.PublicKey)
	if err != nil {
		t.Fatalf("Bob's X25519Exchange failed: %v", err)
	}

	// Verify both computed the same shared secret
	if !bytes.Equal(aliceShared[:], bobShared[:]) {
		t.Error("Shared secrets do not match")
	}
}

// TestDeriveSessionKeys tests session key derivation is symmetric
func TestDeriveSessionKeys(t *testing.T) {
	// Generate keypairs
	alice, err := GenerateX25519()
	if err != nil {
		t.Fatalf("Failed to generate Alice's keypair: %v", err)
	}

	bob, err := GenerateX25519()
	if err != nil {
		t.Fatalf("Failed to generate Bob's keypair: %v", err)
	}

	// Mock manifest hash (32 bytes)
	manifestHash := make([]byte, 32)
	rand.Read(manifestHash)

	// Alice derives session keys
	aliceKeys, err := DeriveSessionKeys(&alice.PrivateKey, &bob.PublicKey, manifestHash)
	if err != nil {
		t.Fatalf("Alice's DeriveSessionKeys failed: %v", err)
	}

	// Bob derives session keys
	bobKeys, err := DeriveSessionKeys(&bob.PrivateKey, &alice.PublicKey, manifestHash)
	if err != nil {
		t.Fatalf("Bob's DeriveSessionKeys failed: %v", err)
	}

	// Verify both derived identical keys
	if !bytes.Equal(aliceKeys.PayloadKey[:], bobKeys.PayloadKey[:]) {
		t.Error("PayloadKeys do not match")
	}

	if !bytes.Equal(aliceKeys.ControlKey[:], bobKeys.ControlKey[:]) {
		t.Error("ControlKeys do not match")
	}

	if !bytes.Equal(aliceKeys.IVBase[:], bobKeys.IVBase[:]) {
		t.Error("IVBases do not match")
	}
}

// TestSealAndOpen tests AES-GCM encryption roundtrip
func TestSealAndOpen(t *testing.T) {
	// Generate random key and nonce
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	rand.Read(key)
	rand.Read(nonce)

	plaintext := []byte("Hello from QuantaraX!")
	aad := []byte("chunk-0")

	// Encrypt
	ciphertext, err := Seal(key, nonce, aad, plaintext)
	if err != nil {
		t.Fatalf("Seal() failed: %v", err)
	}

	// Verify ciphertext is longer (plaintext + 16-byte tag)
	if len(ciphertext) != len(plaintext)+16 {
		t.Errorf("Ciphertext length = %d, want %d", len(ciphertext), len(plaintext)+16)
	}

	// Decrypt
	decrypted, err := Open(key, nonce, aad, ciphertext)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Verify decrypted matches original
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted plaintext does not match original")
	}
}

// TestAuthenticationFailure tests that tampered ciphertext is rejected
func TestAuthenticationFailure(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	rand.Read(key)
	rand.Read(nonce)

	plaintext := []byte("Secret message")
	ciphertext, err := Seal(key, nonce, nil, plaintext)
	if err != nil {
		t.Fatalf("Seal() failed: %v", err)
	}

	// Tamper with ciphertext (flip a bit)
	ciphertext[0] ^= 0x01

	// Attempt to decrypt tampered ciphertext
	_, err = Open(key, nonce, nil, ciphertext)
	if err == nil {
		t.Error("Open() should fail on tampered ciphertext")
	}
}

// TestWrongAAD tests that mismatched AAD causes authentication failure
func TestWrongAAD(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	rand.Read(key)
	rand.Read(nonce)

	plaintext := []byte("Message")
	aad := []byte("chunk-0")

	ciphertext, err := Seal(key, nonce, aad, plaintext)
	if err != nil {
		t.Fatalf("Seal() failed: %v", err)
	}

	// Decrypt with different AAD
	wrongAAD := []byte("chunk-1")
	_, err = Open(key, nonce, wrongAAD, ciphertext)
	if err == nil {
		t.Error("Open() should fail with mismatched AAD")
	}
}

// TestDeriveNonceUniqueness tests nonce uniqueness across 10,000 chunks
func TestDeriveNonceUniqueness(t *testing.T) {
	var ivBase [12]byte
	rand.Read(ivBase[:])

	nonceSet := make(map[[12]byte]bool)
	const numChunks = 10000

	for i := uint32(0); i < numChunks; i++ {
		nonce := DeriveChunkNonce(ivBase, i)

		if nonceSet[nonce] {
			t.Fatalf("Nonce collision detected at chunk %d", i)
		}
		nonceSet[nonce] = true
	}

	t.Logf("Generated %d unique nonces", len(nonceSet))
}

// TestDeriveNonceDeterministic tests nonce derivation is deterministic
func TestDeriveNonceDeterministic(t *testing.T) {
	var ivBase [12]byte
	rand.Read(ivBase[:])

	chunkIndex := uint32(42)

	nonce1 := DeriveChunkNonce(ivBase, chunkIndex)
	nonce2 := DeriveChunkNonce(ivBase, chunkIndex)

	if !bytes.Equal(nonce1[:], nonce2[:]) {
		t.Error("Nonce derivation is not deterministic")
	}
}

// TestControlNonceDistinct tests control nonces are distinct from chunk nonces
func TestControlNonceDistinct(t *testing.T) {
	var ivBase [12]byte
	rand.Read(ivBase[:])

	chunkNonce := DeriveChunkNonce(ivBase, 0)
	controlNonce := DeriveControlNonce(ivBase, 0)

	if bytes.Equal(chunkNonce[:], controlNonce[:]) {
		t.Error("Chunk nonce and control nonce should be different")
	}
}

// TestSaveLoadKeyWithPassphrase tests keystore encryption roundtrip
func TestSaveLoadKeyWithPassphrase(t *testing.T) {
	// Generate Ed25519 keypair
	kp, err := GenerateEd25519()
	if err != nil {
		t.Fatalf("GenerateEd25519() failed: %v", err)
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "identity.key")
	passphrase := "test-passphrase-123"

	// Save with passphrase
	err = SaveKey(kp.PrivateKey, keystorePath, passphrase)
	if err != nil {
		t.Fatalf("SaveKey() failed: %v", err)
	}

	// Load with correct passphrase
	loadedKey, err := LoadKey(keystorePath, passphrase)
	if err != nil {
		t.Fatalf("LoadKey() failed: %v", err)
	}

	// Verify keys match
	if !bytes.Equal(loadedKey, kp.PrivateKey) {
		t.Error("Loaded key does not match original")
	}

	// Test wrong passphrase
	_, err = LoadKey(keystorePath, "wrong-passphrase")
	if err == nil {
		t.Error("LoadKey() should fail with wrong passphrase")
	}
}

// TestSaveLoadKeyWithoutPassphrase tests insecure keystore
func TestSaveLoadKeyWithoutPassphrase(t *testing.T) {
	kp, err := GenerateEd25519()
	if err != nil {
		t.Fatalf("GenerateEd25519() failed: %v", err)
	}

	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "identity.key")

	// Save without passphrase (insecure)
	err = SaveKey(kp.PrivateKey, keystorePath, "")
	if err != nil {
		t.Fatalf("SaveKey() failed: %v", err)
	}

	// Verify .insecure extension was added
	insecurePath := keystorePath + ".insecure"
	if _, err := os.Stat(insecurePath); os.IsNotExist(err) {
		t.Error("Insecure keystore file was not created")
	}

	// Load from insecure keystore
	loadedKey, err := LoadKey(insecurePath, "")
	if err != nil {
		t.Fatalf("LoadKey() failed: %v", err)
	}

	if !bytes.Equal(loadedKey, kp.PrivateKey) {
		t.Error("Loaded key does not match original")
	}
}

// TestChunkEncryptionWorkflow tests realistic chunk encryption scenario
func TestChunkEncryptionWorkflow(t *testing.T) {
	// Setup: Generate keypairs and derive session keys
	alice, _ := GenerateX25519()
	bob, _ := GenerateX25519()

	manifestHash := make([]byte, 32)
	rand.Read(manifestHash)

	aliceKeys, _ := DeriveSessionKeys(&alice.PrivateKey, &bob.PublicKey, manifestHash)
	bobKeys, _ := DeriveSessionKeys(&bob.PrivateKey, &alice.PublicKey, manifestHash)

	// Simulate encrypting 100 chunks
	numChunks := 100
	for i := 0; i < numChunks; i++ {
		chunkData := []byte("chunk data " + string(rune(i)))
		chunkIndex := uint32(i)

		// Alice encrypts chunk
		nonce := DeriveChunkNonce(aliceKeys.IVBase, chunkIndex)
		aad := []byte{byte(chunkIndex)} // Simplified AAD
		ciphertext, err := Seal(aliceKeys.PayloadKey[:], nonce[:], aad, chunkData)
		if err != nil {
			t.Fatalf("Chunk %d encryption failed: %v", i, err)
		}

		// Bob decrypts chunk
		bobNonce := DeriveChunkNonce(bobKeys.IVBase, chunkIndex)
		decrypted, err := Open(bobKeys.PayloadKey[:], bobNonce[:], aad, ciphertext)
		if err != nil {
			t.Fatalf("Chunk %d decryption failed: %v", i, err)
		}

		// Verify data matches
		if !bytes.Equal(decrypted, chunkData) {
			t.Errorf("Chunk %d data mismatch", i)
		}
	}

	t.Logf("Successfully encrypted and decrypted %d chunks", numChunks)
}