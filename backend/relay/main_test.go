package main

import (
	"testing"
	"time"
)

func TestNewRelayService(t *testing.T) {
	config := &RelayConfig{
		ListenAddr:       ":4433",
		MaxConnections:   100,
		ConnTimeout:      30 * time.Second,
		StreamBufferSize: 65536,
		AuthMode:         "none",
		LogLevel:         "info",
	}

	service := NewRelayService(config)
	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config.ListenAddr != ":4433" {
		t.Errorf("Expected listen address :4433, got %s", service.config.ListenAddr)
	}
	if service.config.ConnTimeout != 30*time.Second {
		t.Errorf("Expected conn timeout 30s, got %v", service.config.ConnTimeout)
	}
	if service.config.AuthMode != "none" {
		t.Errorf("Expected auth mode 'none', got %s", service.config.AuthMode)
	}
	if service.config.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got %s", service.config.LogLevel)
	}
}

func TestTokenValidation(t *testing.T) {
	config := &RelayConfig{AuthMode: "token"}
	service := NewRelayService(config)

	// Valid token
	if !service.validateToken("valid-token-12345") {
		t.Error("Expected valid token to pass validation")
	}
	// Invalid tokens
	if service.validateToken("") {
		t.Error("Expected empty token to fail validation")
	}
	if service.validateToken("short") {
		t.Error("Expected short token to fail validation")
	}
}

func TestRelayConfig(t *testing.T) {
	config := &RelayConfig{
		ListenAddr:       ":4433",
		MaxConnections:   500,
		ConnTimeout:      30 * time.Second,
		StreamBufferSize: 65536,
		AuthMode:         "none",
		LogLevel:         "debug",
	}

	if config.MaxConnections != 500 {
		t.Errorf("Expected max connections 500, got %d", config.MaxConnections)
	}
	if config.StreamBufferSize != 65536 {
		t.Errorf("Expected stream buffer size 65536, got %d", config.StreamBufferSize)
	}
	if config.ConnTimeout != 30*time.Second {
		t.Errorf("Expected conn timeout 30s, got %v", config.ConnTimeout)
	}
	if config.ListenAddr == "" {
		t.Errorf("Expected non-empty listen address")
	}
}
