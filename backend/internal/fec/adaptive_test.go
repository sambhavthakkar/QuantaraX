package fec

import (
	"testing"
	"time"
)

func TestAdaptivePolicy_EnableOnHighLoss(t *testing.T) {
	config := DefaultPolicyConfig()
	config.MinObservation = 100 * time.Millisecond // Short for testing
	policy := NewAdaptivePolicy(config)

	// Simulate high loss rate
	for i := 0; i < 10; i++ {
		policy.Update(2.0) // 2% loss
	}

	time.Sleep(150 * time.Millisecond)

	// Update with high loss again to trigger state change
	policy.Update(2.0)

	enabled, k, r := policy.GetParameters()
	if !enabled {
		t.Error("Policy should be enabled with 2% loss")
	}
	if k != 8 {
		t.Errorf("Expected k=8, got k=%d", k)
	}
	if r != 2 {
		t.Errorf("Expected r=2, got r=%d", r)
	}
}

func TestAdaptivePolicy_DisableOnLowLoss(t *testing.T) {
	config := DefaultPolicyConfig()
	config.MinObservation = 50 * time.Millisecond
	policy := NewAdaptivePolicy(config)

	// Enable FEC
	policy.SetEnabled(true)

	// Simulate low loss rate
	for i := 0; i < 10; i++ {
		policy.Update(0.1) // 0.1% loss
	}

	time.Sleep(550 * time.Millisecond) // Longer wait for disable

	// Update with low loss again
	policy.Update(0.1)

	enabled, _, _ := policy.GetParameters()
	if enabled {
		t.Error("Policy should be disabled with 0.1% loss")
	}
}

func TestAdaptivePolicy_AdjustParityShards(t *testing.T) {
	config := DefaultPolicyConfig()
	config.MinObservation = 50 * time.Millisecond
	policy := NewAdaptivePolicy(config)

	// Enable FEC with moderate loss
	policy.SetEnabled(true)

	// Simulate increasing loss rate
	for i := 0; i < 10; i++ {
		policy.Update(6.0) // 6% loss
	}

	time.Sleep(100 * time.Millisecond)
	policy.Update(6.0)

	_, _, r := policy.GetParameters()
	if r < 3 {
		t.Errorf("Expected r >= 3 for high loss, got r=%d", r)
	}
}

func TestAdaptivePolicy_ManualOverride(t *testing.T) {
	policy := NewAdaptivePolicy(DefaultPolicyConfig())

	// Manually enable
	policy.SetEnabled(true)
	enabled, _, _ := policy.GetParameters()
	if !enabled {
		t.Error("Manual enable failed")
	}

	// Manually set parity shards
	err := policy.SetParityShards(3)
	if err != nil {
		t.Fatalf("SetParityShards failed: %v", err)
	}

	_, _, r := policy.GetParameters()
	if r != 3 {
		t.Errorf("Expected r=3, got r=%d", r)
	}
}

func TestAdaptivePolicy_GetState(t *testing.T) {
	policy := NewAdaptivePolicy(DefaultPolicyConfig())

	state := policy.GetState()
	if state.Enabled {
		t.Error("Policy should start disabled")
	}
	if state.K != 8 {
		t.Errorf("Expected K=8, got K=%d", state.K)
	}
}

func TestAdaptivePolicy_Reset(t *testing.T) {
	policy := NewAdaptivePolicy(DefaultPolicyConfig())

	// Modify state
	policy.SetEnabled(true)
	policy.SetParityShards(4)
	for i := 0; i < 10; i++ {
		policy.Update(5.0)
	}

	// Reset
	policy.Reset()

	state := policy.GetState()
	if state.Enabled {
		t.Error("Policy should be disabled after reset")
	}
	if state.R != 2 {
		t.Errorf("Expected R=2 after reset, got R=%d", state.R)
	}
}