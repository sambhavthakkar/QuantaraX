package manager

import (
	"testing"
)

func TestChunkBitmap_SetAndHas(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 100)

	// Test setting a chunk
	err := bitmap.SetChunk(5)
	if err != nil {
		t.Fatalf("SetChunk failed: %v", err)
	}

	// Verify chunk is set
	if !bitmap.HasChunk(5) {
		t.Error("Expected chunk 5 to be set")
	}

	// Verify other chunks are not set
	if bitmap.HasChunk(4) {
		t.Error("Expected chunk 4 to not be set")
	}
}

func TestChunkBitmap_GetMissing(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 10)

	// Set chunks 0, 2, 4, 6, 8
	for i := int64(0); i < 10; i += 2 {
		bitmap.SetChunk(i)
	}

	missing := bitmap.GetMissing()
	expected := []int64{1, 3, 5, 7, 9}

	if len(missing) != len(expected) {
		t.Fatalf("Expected %d missing chunks, got %d", len(expected), len(missing))
	}

	for i, chunk := range expected {
		if missing[i] != chunk {
			t.Errorf("Expected missing chunk %d, got %d", chunk, missing[i])
		}
	}
}

func TestChunkBitmap_IsComplete(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 5)

	if bitmap.IsComplete() {
		t.Error("Empty bitmap should not be complete")
	}

	// Set all chunks
	for i := int64(0); i < 5; i++ {
		bitmap.SetChunk(i)
	}

	if !bitmap.IsComplete() {
		t.Error("Bitmap should be complete after setting all chunks")
	}
}

func TestChunkBitmap_Serialize(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 16)

	// Set chunks 0, 5, 10, 15
	bitmap.SetChunk(0)
	bitmap.SetChunk(5)
	bitmap.SetChunk(10)
	bitmap.SetChunk(15)

	// Serialize
	data := bitmap.Serialize()

	// Create new bitmap and deserialize
	bitmap2 := NewChunkBitmap("test-session-2", 16)
	err := bitmap2.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify chunks match
	for i := int64(0); i < 16; i++ {
		if bitmap.HasChunk(i) != bitmap2.HasChunk(i) {
			t.Errorf("Chunk %d mismatch after deserialize", i)
		}
	}
}

func TestChunkBitmap_GetProgress(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 20)

	// Set 5 chunks
	for i := int64(0); i < 5; i++ {
		bitmap.SetChunk(i)
	}

	received, total := bitmap.GetProgress()
	if received != 5 {
		t.Errorf("Expected 5 received chunks, got %d", received)
	}
	if total != 20 {
		t.Errorf("Expected 20 total chunks, got %d", total)
	}
}

func TestChunkBitmap_OutOfRange(t *testing.T) {
	bitmap := NewChunkBitmap("test-session", 10)

	// Test negative index
	err := bitmap.SetChunk(-1)
	if err == nil {
		t.Error("Expected error for negative chunk index")
	}

	// Test index too large
	err = bitmap.SetChunk(100)
	if err == nil {
		t.Error("Expected error for chunk index out of range")
	}
}
