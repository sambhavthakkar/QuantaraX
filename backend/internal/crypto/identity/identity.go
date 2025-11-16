package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultPaths returns default private and public key paths in ~/.quantarax
func DefaultPaths() (privPath, pubPath string, err error) {
	h, err := os.UserHomeDir()
	if err != nil { return "", "", err }
	dir := filepath.Join(h, ".quantarax")
	return filepath.Join(dir, "id_ed25519"), filepath.Join(dir, "id_ed25519.pub"), nil
}

// LoadOrCreate loads ed25519 keys from given priv path (and pub path = priv+".pub" if pubPath empty). Generates if missing.
func LoadOrCreate(privPath, pubPath string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if privPath == "" {
		p, u, err := DefaultPaths(); if err != nil { return nil, nil, err }
		privPath, pubPath = p, u
	}
	if pubPath == "" { pubPath = privPath + ".pub" }

	priv, pub, err := load(privPath, pubPath)
	if err == nil { return priv, pub, nil }
	if !errors.Is(err, fs.ErrNotExist) {
		// If load failed for other reasons, return error
		return nil, nil, err
	}
	// Generate
	if err := os.MkdirAll(filepath.Dir(privPath), 0o700); err != nil { return nil, nil, err }
	pub, priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil { return nil, nil, err }
	if err := writeKeyFiles(privPath, pubPath, priv, pub); err != nil { return nil, nil, err }
	return priv, pub, nil
}

func load(privPath, pubPath string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pbytes, err := os.ReadFile(privPath)
	if err != nil { return nil, nil, err }
	ubytes, err := os.ReadFile(pubPath)
	if err != nil { return nil, nil, err }
	priv, err := decodeKey(pbytes)
	if err != nil { return nil, nil, fmt.Errorf("invalid private key: %w", err) }
	pub, err := decodePub(ubytes)
	if err != nil { return nil, nil, fmt.Errorf("invalid public key: %w", err) }
	if len(priv) != ed25519.PrivateKeySize || len(pub) != ed25519.PublicKeySize { return nil, nil, fmt.Errorf("bad key sizes") }
	return priv, pub, nil
}

func writeKeyFiles(privPath, pubPath string, priv ed25519.PrivateKey, pub ed25519.PublicKey) error {
	encPriv := encodeKey(priv)
	encPub := encodePub(pub)
	if err := os.WriteFile(privPath, encPriv, 0o600); err != nil { return err }
	if err := os.WriteFile(pubPath, encPub, 0o644); err != nil { return err }
	return nil
}

func encodeKey(k ed25519.PrivateKey) []byte { return []byte(base64.StdEncoding.EncodeToString(k)) }
func encodePub(k ed25519.PublicKey) []byte { return []byte(base64.StdEncoding.EncodeToString(k)) }

func decodeKey(b []byte) (ed25519.PrivateKey, error) {
	dec, err := base64.StdEncoding.DecodeString(string(bytesTrimSpace(b)))
	if err != nil { return nil, err }
	return ed25519.PrivateKey(dec), nil
}
func decodePub(b []byte) (ed25519.PublicKey, error) {
	dec, err := base64.StdEncoding.DecodeString(string(bytesTrimSpace(b)))
	if err != nil { return nil, err }
	return ed25519.PublicKey(dec), nil
}

func bytesTrimSpace(b []byte) []byte {
	i := 0; j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') { i++ }
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') { j-- }
	return b[i:j]
}
