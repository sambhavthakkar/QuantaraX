package config

import (
	"os"
	"path/filepath"
)

// Config holds daemon configuration
type Config struct {
	GRPCAddress            string
	RESTAddress            string
	QUICAddress            string
	KeysDirectory          string
	ChunkSize              int64
	MaxConcurrentTransfers int
	TokenTTL               int
	EventBufferSize        int
	WorkerCount            int
	QueueDepth             int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	keysDir := filepath.Join(homeDir, ".local", "share", "quantarax", "keys")

	return &Config{
		GRPCAddress:            "127.0.0.1:9090",
		RESTAddress:            "127.0.0.1:8080",
		QUICAddress:            ":4433",
		KeysDirectory:          keysDir,
		ChunkSize:              1048576, // 1 MiB
		MaxConcurrentTransfers: 10,
		TokenTTL:               3600,
		EventBufferSize:        100,
		WorkerCount:            8,
		QueueDepth:             32,
	}
}

// LoadConfig loads configuration from file (simplified - just returns default)
func LoadConfig(configPath string) (*Config, error) {
	// For simplicity, return default config
	// In production, this would parse YAML file
	return DefaultConfig(), nil
}
