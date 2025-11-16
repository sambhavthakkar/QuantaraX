package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
	"net/http/pprof"
)

// TokenEntry represents a registered transfer token
type TokenEntry struct {
	Token              string    `json:"token"`
	EphemeralPublicKey string    `json:"ephemeral_public_key"`
	ManifestHash       string    `json:"manifest_hash"`
	RelayHints         []string  `json:"relay_hints"`
	SenderAddress      string    `json:"sender_address,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	RegistrationID     string    `json:"registration_id"`
}

// UserEntry represents a registered username
type UserEntry struct {
	Username      string    `json:"username"`
	PublicKey     string    `json:"public_key"`
	Fingerprint   string    `json:"fingerprint"`
	RelayHints    []string  `json:"relay_hints,omitempty"`
	DirectAddress string    `json:"direct_address,omitempty"`
	RegisteredAt  time.Time `json:"registered_at"`
	LastSeen      time.Time `json:"last_seen"`
}

// TokenRegistry manages transfer tokens
type TokenRegistry struct {
	entries map[string]*TokenEntry
	mu      sync.RWMutex
}

// UsernameRegistry manages username registrations
type UsernameRegistry struct {
	entries map[string]*UserEntry
	mu      sync.RWMutex
}

// BootstrapService manages the bootstrap discovery service
type BootstrapService struct {
	tokens    *TokenRegistry
	usernames *UsernameRegistry
	limiters  map[string]*rate.Limiter
	limiterMu sync.RWMutex
	maxTTL    time.Duration
}

func NewTokenRegistry() *TokenRegistry {
	return &TokenRegistry{
		entries: make(map[string]*TokenEntry),
	}
}

func NewUsernameRegistry() *UsernameRegistry {
	return &UsernameRegistry{
		entries: make(map[string]*UserEntry),
	}
}

func NewBootstrapService(maxTTL time.Duration) *BootstrapService {
	return &BootstrapService{
		tokens:    NewTokenRegistry(),
		usernames: NewUsernameRegistry(),
		limiters:  make(map[string]*rate.Limiter),
		maxTTL:    maxTTL,
	}
}

// RegisterToken registers a new transfer token
func (tr *TokenRegistry) RegisterToken(entry *TokenEntry) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.entries[entry.Token]; exists {
		return fmt.Errorf("token already exists")
	}

	tr.entries[entry.Token] = entry
	return nil
}

// LookupToken retrieves a token entry
func (tr *TokenRegistry) LookupToken(token string) (*TokenEntry, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	entry, exists := tr.entries[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return entry, nil
}

// CleanupExpired removes expired tokens
func (tr *TokenRegistry) CleanupExpired() int {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	count := 0
	now := time.Now()
	for token, entry := range tr.entries {
		if now.After(entry.ExpiresAt) {
			delete(tr.entries, token)
			count++
		}
	}
	return count
}

// Count returns the number of active tokens
func (tr *TokenRegistry) Count() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return len(tr.entries)
}

// RegisterUser registers a new username
func (ur *UsernameRegistry) RegisterUser(entry *UserEntry) error {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, exists := ur.entries[entry.Username]; exists {
		return fmt.Errorf("username already taken")
	}

	ur.entries[entry.Username] = entry
	return nil
}

// LookupUser retrieves a user entry
func (ur *UsernameRegistry) LookupUser(username string) (*UserEntry, error) {
	ur.mu.RLock()
	defer ur.mu.RUnlock()

	entry, exists := ur.entries[username]
	if !exists {
		return nil, fmt.Errorf("username not found")
	}

	return entry, nil
}

// Count returns the number of registered users
func (ur *UsernameRegistry) Count() int {
	ur.mu.RLock()
	defer ur.mu.RUnlock()
	return len(ur.entries)
}

// Rate limiter
func (bs *BootstrapService) getRateLimiter(ip string, limit rate.Limit, burst int) *rate.Limiter {
	bs.limiterMu.Lock()
	defer bs.limiterMu.Unlock()

	limiter, exists := bs.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(limit, burst)
		bs.limiters[ip] = limiter
	}
	return limiter
}

// HTTP Handlers

func (bs *BootstrapService) handleRegisterToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	ip := getClientIP(r)
	limiter := bs.getRateLimiter(ip, rate.Limit(20.0/60.0), 20) // 20 per minute
	if !limiter.Allow() {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Token              string   `json:"token"`
		EphemeralPublicKey string   `json:"ephemeral_public_key"`
		ManifestHash       string   `json:"manifest_hash"`
		RelayHints         []string `json:"relay_hints"`
		SenderAddress      string   `json:"sender_address"`
		TTLSeconds         int      `json:"ttl_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Token == "" || req.EphemeralPublicKey == "" || req.ManifestHash == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Set default TTL
	if req.TTLSeconds == 0 {
		req.TTLSeconds = 3600 // 1 hour default
	}

	// Enforce max TTL
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl > bs.maxTTL {
		ttl = bs.maxTTL
	}

	entry := &TokenEntry{
		Token:              req.Token,
		EphemeralPublicKey: req.EphemeralPublicKey,
		ManifestHash:       req.ManifestHash,
		RelayHints:         req.RelayHints,
		SenderAddress:      req.SenderAddress,
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(ttl),
		RegistrationID:     uuid.New().String(),
	}

	if err := bs.tokens.RegisterToken(entry); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("Token registered: %s (expires: %s)", entry.Token, entry.ExpiresAt.Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":           entry.Token,
		"expires_at":      entry.ExpiresAt.Format(time.RFC3339),
		"registration_id": entry.RegistrationID,
	})
}

func (bs *BootstrapService) handleLookupToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	ip := getClientIP(r)
	limiter := bs.getRateLimiter(ip, rate.Limit(200.0/60.0), 200) // 200 per minute
	if !limiter.Allow() {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	token := r.URL.Path[len("/api/v1/token/"):]
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	entry, err := bs.tokens.LookupToken(token)
	if err != nil {
		http.Error(w, "Token not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (bs *BootstrapService) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	ip := getClientIP(r)
	limiter := bs.getRateLimiter(ip, rate.Limit(5.0/3600.0), 5) // 5 per hour
	if !limiter.Allow() {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username      string   `json:"username"`
		PublicKey     string   `json:"public_key"`
		RelayHints    []string `json:"relay_hints"`
		DirectAddress string   `json:"direct_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate username
	if !isValidUsername(req.Username) {
		http.Error(w, "Invalid username format", http.StatusBadRequest)
		return
	}

	if req.PublicKey == "" {
		http.Error(w, "Public key required", http.StatusBadRequest)
		return
	}

	entry := &UserEntry{
		Username:      req.Username,
		PublicKey:     req.PublicKey,
		Fingerprint:   computeFingerprint(req.PublicKey),
		RelayHints:    req.RelayHints,
		DirectAddress: req.DirectAddress,
		RegisteredAt:  time.Now(),
		LastSeen:      time.Now(),
	}

	if err := bs.usernames.RegisterUser(entry); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("User registered: %s (fingerprint: %s)", entry.Username, entry.Fingerprint)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":      entry.Username,
		"fingerprint":   entry.Fingerprint,
		"registered_at": entry.RegisteredAt.Format(time.RFC3339),
	})
}

func (bs *BootstrapService) handleLookupUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	ip := getClientIP(r)
	limiter := bs.getRateLimiter(ip, rate.Limit(100.0/60.0), 100) // 100 per minute
	if !limiter.Allow() {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	username := r.URL.Path[len("/api/v1/lookup/"):]
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	entry, err := bs.usernames.LookupUser(username)
	if err != nil {
		http.Error(w, "Username not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (bs *BootstrapService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "healthy",
		"token_count":    bs.tokens.Count(),
		"username_count": bs.usernames.Count(),
	})
}

// Helper functions

func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	// Check reserved names
	reserved := []string{"admin", "root", "system", "quantarax"}
	for _, r := range reserved {
		if username == r {
			return false
		}
	}
	return true
}

func computeFingerprint(publicKey string) string {
	// Simple hash for fingerprint (first 16 chars)
	if len(publicKey) > 16 {
		return publicKey[:16]
	}
	return publicKey
}

func main() {
	listen := flag.String("listen", ":8082", "HTTP listen address")
	logLevel := flag.String("log-level", "info", "Logging level")
	maxTTL := flag.Duration("token-ttl-max", 24*time.Hour, "Maximum token TTL")
	cleanupInterval := flag.Duration("cleanup-interval", 60*time.Second, "Cleanup interval")
	flag.Parse()

	log.Printf("QuantaraX Bootstrap Service starting...")
	log.Printf("Log level: %s", *logLevel)
	log.Printf("Max token TTL: %s", *maxTTL)
	log.Printf("Cleanup interval: %s", *cleanupInterval)

// Basic address validation
if *listen == "" { log.Fatalf("listen address cannot be empty") }
	service := NewBootstrapService(*maxTTL)

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(*cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			count := service.tokens.CleanupExpired()
			if count > 0 {
				log.Printf("Cleaned up %d expired tokens", count)
			}
		}
	}()

	// Setup HTTP routes
	http.HandleFunc("/api/v1/token", service.handleRegisterToken)
	http.HandleFunc("/api/v1/token/", service.handleLookupToken)
	http.HandleFunc("/api/v1/register", service.handleRegisterUser)
	http.HandleFunc("/api/v1/lookup/", service.handleLookupUser)
http.HandleFunc("/health", service.handleHealth)
	// pprof endpoints
	http.HandleFunc("/debug/pprof/", pprof.Index)
	http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	http.HandleFunc("/debug/pprof/profile", pprof.Profile)
	http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	http.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Start HTTP server
	server := &http.Server{
		Addr:         *listen,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Bootstrap service listening on %s", *listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	log.Printf("Final stats - Tokens: %d, Users: %d", service.tokens.Count(), service.usernames.Count())
}
