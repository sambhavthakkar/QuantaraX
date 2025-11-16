package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenRegistration(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	reqBody := map[string]interface{}{
		"token":                "test-token-123",
		"ephemeral_public_key": "test-key",
		"manifest_hash":        "test-hash",
		"relay_hints":          []string{"relay1.test:4433"},
		"ttl_seconds":          3600,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	service.handleRegisterToken(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["token"] != "test-token-123" {
		t.Errorf("Expected token test-token-123, got %v", resp["token"])
	}
}

func TestTokenLookup(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	// Register a token first
	entry := &TokenEntry{
		Token:              "lookup-test",
		EphemeralPublicKey: "key123",
		ManifestHash:       "hash123",
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(1 * time.Hour),
		RegistrationID:     "reg123",
	}
	service.tokens.RegisterToken(entry)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/lookup-test", nil)
	w := httptest.NewRecorder()

	service.handleLookupToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp TokenEntry
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Token != "lookup-test" {
		t.Errorf("Expected token lookup-test, got %s", resp.Token)
	}
}

func TestTokenExpiration(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	// Register an expired token
	entry := &TokenEntry{
		Token:              "expired-test",
		EphemeralPublicKey: "key123",
		ManifestHash:       "hash123",
		CreatedAt:          time.Now().Add(-2 * time.Hour),
		ExpiresAt:          time.Now().Add(-1 * time.Hour),
		RegistrationID:     "reg123",
	}
	service.tokens.RegisterToken(entry)

	// Try to lookup expired token
	_, err := service.tokens.LookupToken("expired-test")
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestDuplicateToken(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	entry := &TokenEntry{
		Token:              "dup-test",
		EphemeralPublicKey: "key123",
		ManifestHash:       "hash123",
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(1 * time.Hour),
		RegistrationID:     "reg123",
	}

	err := service.tokens.RegisterToken(entry)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	err = service.tokens.RegisterToken(entry)
	if err == nil {
		t.Error("Expected error for duplicate token, got nil")
	}
}

func TestUsernameRegistration(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	reqBody := map[string]interface{}{
		"username":   "testuser",
		"public_key": "pubkey123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	service.handleRegisterUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestUsernameLookup(t *testing.T) {
	service := NewBootstrapService(24 * time.Hour)

	entry := &UserEntry{
		Username:     "lookupuser",
		PublicKey:    "key123",
		Fingerprint:  "fp123",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	service.usernames.RegisterUser(entry)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lookup/lookupuser", nil)
	w := httptest.NewRecorder()

	service.handleLookupUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestInvalidUsername(t *testing.T) {
	invalidNames := []string{"ab", "verylongusernamethatisinvalid12345678", "user@name", "admin"}

	for _, name := range invalidNames {
		if isValidUsername(name) {
			t.Errorf("Username %s should be invalid", name)
		}
	}

	validNames := []string{"user123", "test_user", "ValidUser"}
	for _, name := range validNames {
		if !isValidUsername(name) {
			t.Errorf("Username %s should be valid", name)
		}
	}
}

func TestCleanupExpired(t *testing.T) {
	registry := NewTokenRegistry()

	// Add some expired and active tokens
	registry.RegisterToken(&TokenEntry{
		Token:     "active",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	registry.RegisterToken(&TokenEntry{
		Token:     "expired1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	registry.RegisterToken(&TokenEntry{
		Token:     "expired2",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	count := registry.CleanupExpired()
	if count != 2 {
		t.Errorf("Expected 2 tokens cleaned up, got %d", count)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected 1 active token, got %d", registry.Count())
	}
}
