package fec

import (
	"bytes"
	"testing"
)

func TestFEC_EncodeDecode(t *testing.T) {
	k, r := 8, 2
	dataShards := make([][]byte, k)
	
	// Create test data
	for i := range dataShards {
		dataShards[i] = make([]byte, 1024)
		for j := range dataShards[i] {
			dataShards[i][j] = byte(i)
		}
	}

	// Encode
	encoder, err := NewEncoder(k, r)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	parityShards, err := encoder.Encode(dataShards)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	if len(parityShards) != r {
		t.Fatalf("Expected %d parity shards, got %d", r, len(parityShards))
	}

	// Simulate losing 2 shards
	allShards := make([][]byte, k+r)
	copy(allShards[:k], dataShards)
	copy(allShards[k:], parityShards)

	// Mark shards 3 and 7 as lost
	allShards[3] = nil
	allShards[7] = nil

	// Decode
	decoder, err := NewDecoder(k, r)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	err = decoder.Reconstruct(allShards)
	if err != nil {
		t.Fatalf("Reconstruction failed: %v", err)
	}

	// Verify reconstructed data
	if !bytes.Equal(allShards[3], dataShards[3]) {
		t.Error("Reconstructed shard 3 does not match original")
	}
	if !bytes.Equal(allShards[7], dataShards[7]) {
		t.Error("Reconstructed shard 7 does not match original")
	}
}

func TestFEC_TooManyLost(t *testing.T) {
	k, r := 8, 2
	dataShards := make([][]byte, k)

	for i := range dataShards {
		dataShards[i] = make([]byte, 1024)
	}

	encoder, _ := NewEncoder(k, r)
	parityShards, _ := encoder.Encode(dataShards)

	allShards := make([][]byte, k+r)
	copy(allShards[:k], dataShards)
	copy(allShards[k:], parityShards)

	// Mark 3 shards as lost (more than r=2)
	allShards[1] = nil
	allShards[3] = nil
	allShards[7] = nil

	decoder, _ := NewDecoder(k, r)
	err := decoder.Reconstruct(allShards)
	if err == nil {
		t.Error("Expected error when too many shards are lost")
	}
}

func TestFEC_NoMissing(t *testing.T) {
	k, r := 8, 2
	dataShards := make([][]byte, k)

	for i := range dataShards {
		dataShards[i] = make([]byte, 1024)
	}

	encoder, _ := NewEncoder(k, r)
	parityShards, _ := encoder.Encode(dataShards)

	allShards := make([][]byte, k+r)
	copy(allShards[:k], dataShards)
	copy(allShards[k:], parityShards)

	decoder, _ := NewDecoder(k, r)
	err := decoder.Reconstruct(allShards)
	if err != nil {
		t.Errorf("Reconstruction should succeed with no missing shards: %v", err)
	}
}

func TestFEC_InvalidParameters(t *testing.T) {
	// Test invalid K
	_, err := NewEncoder(0, 2)
	if err == nil {
		t.Error("Expected error for k=0")
	}

	_, err = NewEncoder(300, 2)
	if err == nil {
		t.Error("Expected error for k=300")
	}

	// Test invalid R
	_, err = NewEncoder(8, 0)
	if err == nil {
		t.Error("Expected error for r=0")
	}

	_, err = NewEncoder(8, 300)
	if err == nil {
		t.Error("Expected error for r=300")
	}
}
