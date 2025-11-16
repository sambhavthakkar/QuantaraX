package handshake

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/quantarax/backend/internal/crypto"
	"golang.org/x/crypto/hkdf"
)

type ClientHello struct {
	Type        string `json:"type"`
	SessionID   string `json:"session_id"`
	ClientEph   string `json:"client_eph_pub"` // base64
	ClientIDPub string `json:"client_id_pub"`  // base64 (ed25519)
	Sig         string `json:"sig,omitempty"`  // base64 (ed25519 over transcript)
	TokenHMAC   string `json:"token_hmac,omitempty"`
}

type ServerHello struct {
	Type       string `json:"type"`
	ServerEph  string `json:"server_eph_pub"`
	ServerID   string `json:"server_id_pub"`
	Sig        string `json:"sig,omitempty"`
}

type SessionKeys struct {
	PayloadKey [32]byte
	IVBase     [12]byte
}

func serialize(v any) []byte { b, _ := json.Marshal(v); return b }

func sign(priv ed25519.PrivateKey, parts ...[]byte) (string, error) {
	msg := []byte("QX-HANDSHAKE|")
	for i, p := range parts { msg = append(msg, p...); if i+1 < len(parts) { msg = append(msg, '|') } }
	sig := ed25519.Sign(priv, msg)
	return base64.StdEncoding.EncodeToString(sig), nil
}

func verify(pub ed25519.PublicKey, sigb64 string, parts ...[]byte) bool {
	msg := []byte("QX-HANDSHAKE|")
	for i, p := range parts { msg = append(msg, p...); if i+1 < len(parts) { msg = append(msg, '|') } }
	sig, err := base64.StdEncoding.DecodeString(sigb64)
	if err != nil { return false }
	return ed25519.Verify(pub, msg, sig)
}

// Derive session keys using HKDF-SHA256 over ECDH + transcript hash
func deriveKeys(shared []byte, transcript []byte) (SessionKeys, error) {
	salt := sha256.Sum256(transcript)
	h := hkdf.New(sha256.New, shared, salt[:], []byte("quantarax-session-keys"))
	var out [48]byte
	if _, err := io.ReadFull(h, out[:]); err != nil { return SessionKeys{}, err }
	var sk SessionKeys
	copy(sk.PayloadKey[:], out[:32])
	copy(sk.IVBase[:], out[32:44])
	return sk, nil
}

// Compute HMAC binding to optional token secret
func computeTokenHMAC(secret []byte, transcript []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write(transcript)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Client performs handshake on provided io.ReadWriter (stream)
func ClientHandshake(rw io.ReadWriter, sessionID string, clientIDPriv ed25519.PrivateKey, clientIDPub ed25519.PublicKey, tokenSecret []byte) (SessionKeys, error) {
	// generate ephemeral X25519
	kp, err := crypto.GenerateX25519()
	if err != nil { return SessionKeys{}, err }
	clientEphB64 := base64.StdEncoding.EncodeToString(kp.PublicKey[:])
	clientIDB64 := base64.StdEncoding.EncodeToString(clientIDPub)
	ch := ClientHello{Type:"client_hello", SessionID: sessionID, ClientEph: clientEphB64, ClientIDPub: clientIDB64}
	// sign transcript so far
	sig, err := sign(clientIDPriv, []byte("client"), []byte(sessionID), []byte(clientEphB64), []byte(clientIDB64))
	if err == nil { ch.Sig = sig }
	// optional token binding
	transcript := serialize(ch)
	if len(tokenSecret) > 0 { ch.TokenHMAC = computeTokenHMAC(tokenSecret, transcript) }
	// send
	enc := json.NewEncoder(rw)
	if err := enc.Encode(&ch); err != nil { return SessionKeys{}, err }
	// receive server hello
	dec := json.NewDecoder(rw)
	var sh ServerHello
	if err := dec.Decode(&sh); err != nil { return SessionKeys{}, err }
	if sh.Type != "server_hello" { return SessionKeys{}, fmt.Errorf("unexpected msg: %s", sh.Type) }
	// verify server sig if present
	srvPubB, _ := base64.StdEncoding.DecodeString(sh.ServerID)
	if sh.Sig != "" && len(srvPubB) == ed25519.PublicKeySize {
		ok := verify(ed25519.PublicKey(srvPubB), sh.Sig, []byte("server"), []byte(sessionID), []byte(sh.ServerEph), []byte(sh.ServerID))
		if !ok { return SessionKeys{}, fmt.Errorf("server signature invalid") }
	}
	// derive shared
	srvEphB, _ := base64.StdEncoding.DecodeString(sh.ServerEph)
	if len(srvEphB) != 32 { return SessionKeys{}, fmt.Errorf("bad server eph") }
	var srvEph [32]byte; copy(srvEph[:], srvEphB)
	shared := crypto.SharedSecret(&kp.PrivateKey, &srvEph)
	// derive keys
	transcriptB := append(transcript, serialize(sh)...)
	return deriveKeys(shared[:], transcriptB)
}

// Server performs handshake and returns session keys
func ServerHandshake(rw io.ReadWriter, sessionID string, serverIDPriv ed25519.PrivateKey, serverIDPub ed25519.PublicKey, tokenSecret []byte) (SessionKeys, error) {
	dec := json.NewDecoder(rw)
	var ch ClientHello
	if err := dec.Decode(&ch); err != nil { return SessionKeys{}, err }
	if ch.Type != "client_hello" { return SessionKeys{}, fmt.Errorf("unexpected msg: %s", ch.Type) }
	if ch.SessionID != sessionID { return SessionKeys{}, fmt.Errorf("session id mismatch") }
	// verify client sig if present
	cliPubB, _ := base64.StdEncoding.DecodeString(ch.ClientIDPub)
	if ch.Sig != "" && len(cliPubB) == ed25519.PublicKeySize {
		ok := verify(ed25519.PublicKey(cliPubB), ch.Sig, []byte("client"), []byte(ch.SessionID), []byte(ch.ClientEph), []byte(ch.ClientIDPub))
		if !ok { return SessionKeys{}, fmt.Errorf("client signature invalid") }
	}
	// optional token binding check (best-effort)
	transcript := serialize(ch)
	if len(tokenSecret) > 0 && ch.TokenHMAC != "" {
		expected := computeTokenHMAC(tokenSecret, transcript)
		if !strings.EqualFold(expected, ch.TokenHMAC) { return SessionKeys{}, fmt.Errorf("token binding invalid") }
	}
	// generate server eph
	kp, err := crypto.GenerateX25519(); if err != nil { return SessionKeys{}, err }
	srvEphB64 := base64.StdEncoding.EncodeToString(kp.PublicKey[:])
	srvIDB64 := base64.StdEncoding.EncodeToString(serverIDPub)
	sh := ServerHello{Type:"server_hello", ServerEph: srvEphB64, ServerID: srvIDB64}
	sig, err := sign(serverIDPriv, []byte("server"), []byte(ch.SessionID), []byte(srvEphB64), []byte(srvIDB64))
	if err == nil { sh.Sig = sig }
	// send
	enc := json.NewEncoder(rw)
	if err := enc.Encode(&sh); err != nil { return SessionKeys{}, err }
	// derive shared
	cliEphB, _ := base64.StdEncoding.DecodeString(ch.ClientEph)
	if len(cliEphB) != 32 { return SessionKeys{}, fmt.Errorf("bad client eph") }
	var cliEph [32]byte; copy(cliEph[:], cliEphB)
	shared := crypto.SharedSecret(&kp.PrivateKey, &cliEph)
	// derive keys
	transcriptB := append(transcript, serialize(sh)...)
	return deriveKeys(shared[:], transcriptB)
}
