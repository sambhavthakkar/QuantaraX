package fec

import (
	"fmt"
	"sync"
	"time"
)

// PolicyState represents the current FEC policy state
type PolicyState struct {
	Enabled   bool
	K         int     // Data shards
	R         int     // Parity shards
	LossRate  float64 // Current loss rate percentage
	UpdatedAt time.Time
}

// AdaptivePolicy manages FEC parameters based on network conditions
type AdaptivePolicy struct {
	// Configuration
	enableThreshold  float64       // Loss rate to enable FEC (%)
	disableThreshold float64       // Loss rate to disable FEC (%)
	minObservation   time.Duration // Minimum observation time before changes
	defaultK         int           // Default data shards
	defaultR         int           // Default parity shards
	maxR             int           // Maximum parity shards

	// State
	enabled          bool
	currentK         int
	currentR         int
	lossRateSamples  []float64
	lastStateChange  time.Time
	sampleStartTime  time.Time

	mu sync.RWMutex
}

// PolicyConfig holds adaptive policy configuration
type PolicyConfig struct {
	EnableThreshold  float64       // Default: 1.0%
	DisableThreshold float64       // Default: 0.5%
	MinObservation   time.Duration // Default: 30s
	DefaultK         int           // Default: 8
	DefaultR         int           // Default: 2
	MaxR             int           // Default: 4
}

// DefaultPolicyConfig returns default policy configuration
func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{
		EnableThreshold:  1.0,
		DisableThreshold: 0.5,
		MinObservation:   30 * time.Second,
		DefaultK:         8,
		DefaultR:         2,
		MaxR:             4,
	}
}

// NewAdaptivePolicy creates a new adaptive FEC policy
func NewAdaptivePolicy(config PolicyConfig) *AdaptivePolicy {
	return &AdaptivePolicy{
		enableThreshold:  config.EnableThreshold,
		disableThreshold: config.DisableThreshold,
		minObservation:   config.MinObservation,
		defaultK:         config.DefaultK,
		defaultR:         config.DefaultR,
		maxR:             config.MaxR,
		enabled:          false,
		currentK:         config.DefaultK,
		currentR:         config.DefaultR,
		lossRateSamples:  make([]float64, 0, 60), // 60 samples max
		lastStateChange:  time.Now(),
		sampleStartTime:  time.Now(),
	}
}

// Update updates the policy with the latest loss rate
func (ap *AdaptivePolicy) Update(lossRate float64) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Add sample
	ap.lossRateSamples = append(ap.lossRateSamples, lossRate)
	
	// Keep only last 60 samples (10 minutes at 10-second intervals)
	if len(ap.lossRateSamples) > 60 {
		ap.lossRateSamples = ap.lossRateSamples[1:]
	}

	// Calculate average loss rate
	avgLoss := ap.calculateAverageLoss()

	// Check if enough time has passed since last state change
	timeSinceChange := time.Since(ap.lastStateChange)
	if timeSinceChange < ap.minObservation {
		return // Too soon to change state
	}

	// Apply policy rules
	if !ap.enabled && avgLoss > ap.enableThreshold {
		// Enable FEC
		ap.enabled = true
		ap.currentR = ap.defaultR
		ap.lastStateChange = time.Now()
	} else if ap.enabled && avgLoss < ap.disableThreshold {
		// Disable FEC (only after longer observation)
		if timeSinceChange >= ap.minObservation*10 { // 5 minutes
			ap.enabled = false
			ap.lastStateChange = time.Now()
		}
	} else if ap.enabled {
		// Adjust R based on loss rate
		if avgLoss > 5.0 && ap.currentR < ap.maxR {
			ap.currentR = 4
			ap.lastStateChange = time.Now()
		} else if avgLoss > 3.0 && ap.currentR < 3 {
			ap.currentR = 3
			ap.lastStateChange = time.Now()
		} else if avgLoss < 2.0 && ap.currentR > ap.defaultR {
			ap.currentR = ap.defaultR
			ap.lastStateChange = time.Now()
		}
	}
}

// GetParameters returns current FEC parameters
func (ap *AdaptivePolicy) GetParameters() (enabled bool, k, r int) {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.enabled, ap.currentK, ap.currentR
}

// GetState returns current policy state
func (ap *AdaptivePolicy) GetState() PolicyState {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	return PolicyState{
		Enabled:   ap.enabled,
		K:         ap.currentK,
		R:         ap.currentR,
		LossRate:  ap.calculateAverageLoss(),
		UpdatedAt: time.Now(),
	}
}

// SetEnabled manually enables or disables FEC
func (ap *AdaptivePolicy) SetEnabled(enabled bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.enabled = enabled
	ap.lastStateChange = time.Now()
}

// SetParityShards manually sets the number of parity shards
func (ap *AdaptivePolicy) SetParityShards(r int) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if r < 1 || r > ap.maxR {
		return ErrInvalidParityShards
	}

	ap.currentR = r
	ap.lastStateChange = time.Now()
	return nil
}

// calculateAverageLoss calculates exponential moving average of loss rate
func (ap *AdaptivePolicy) calculateAverageLoss() float64 {
	if len(ap.lossRateSamples) == 0 {
		return 0
	}

	// Use exponential moving average with alpha=0.3
	alpha := 0.3
	ema := ap.lossRateSamples[0]
	
	for i := 1; i < len(ap.lossRateSamples); i++ {
		ema = alpha*ap.lossRateSamples[i] + (1-alpha)*ema
	}

	return ema
}

// Reset resets the policy to initial state
func (ap *AdaptivePolicy) Reset() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.enabled = false
	ap.currentR = ap.defaultR
	ap.lossRateSamples = make([]float64, 0, 60)
	ap.lastStateChange = time.Now()
	ap.sampleStartTime = time.Now()
}

var (
	ErrInvalidParityShards = fmt.Errorf("invalid number of parity shards")
)
