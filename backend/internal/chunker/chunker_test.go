package chunker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeManifest_SmallFile(t *testing.T) {
	// Create a small test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "small.bin")
	
	testData := []byte("Hello, QuantaraX!")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute manifest with default chunk size
	opts := DefaultChunkOptions()
	manifest, err := ComputeManifest(testFile, opts)
	if err != nil {
		t.Fatalf("ComputeManifest failed: %v", err)
	}

	// Verify basic properties
	if manifest.ChunkCount != 1 {
		t.Errorf("Expected 1 chunk, got %d", manifest.ChunkCount)
	}
	if manifest.FileSize != int64(len(testData)) {
		t.Errorf("Expected file size %d, got %d", len(testData), manifest.FileSize)
	}
	if manifest.FileName != "small.bin" {
		t.Errorf("Expected filename 'small.bin', got %s", manifest.FileName)
	}
	if manifest.HashAlgo != "BLAKE3" {
		t.Errorf("Expected hash algorithm 'BLAKE3', got %s", manifest.HashAlgo)
	}
	if len(manifest.Chunks) != 1 {
		t.Errorf("Expected 1 chunk descriptor, got %d", len(manifest.Chunks))
	}
	if manifest.Chunks[0].Length != len(testData) {
		t.Errorf("Expected chunk length %d, got %d", len(testData), manifest.Chunks[0].Length)
	}
	if manifest.MerkleRoot == "" {
		t.Error("Merkle root should not be empty")
	}
}

func TestComputeManifest_MultipleChunks(t *testing.T) {
	// Create a file with multiple chunks
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multi.bin")
	
	// Create 2.5 MB file (will be 3 chunks at 1 MB each)
	chunkSize := 1024 * 1024 // 1 MB
	testData := make([]byte, chunkSize*2+chunkSize/2)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute manifest
	opts := ChunkOptions{ChunkSize: chunkSize}
	manifest, err := ComputeManifest(testFile, opts)
	if err != nil {
		t.Fatalf("ComputeManifest failed: %v", err)
	}

	// Verify chunk count
	if manifest.ChunkCount != 3 {
		t.Errorf("Expected 3 chunks, got %d", manifest.ChunkCount)
	}

	// Verify first two chunks are full size
	if manifest.Chunks[0].Length != chunkSize {
		t.Errorf("Chunk 0 expected length %d, got %d", chunkSize, manifest.Chunks[0].Length)
	}
	if manifest.Chunks[1].Length != chunkSize {
		t.Errorf("Chunk 1 expected length %d, got %d", chunkSize, manifest.Chunks[1].Length)
	}

	// Verify last chunk is partial
	if manifest.Chunks[2].Length != chunkSize/2 {
		t.Errorf("Chunk 2 expected length %d, got %d", chunkSize/2, manifest.Chunks[2].Length)
	}
}

func TestComputeManifest_Deterministic(t *testing.T) {
	// Test that same file produces same hashes
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "deterministic.bin")
	
	testData := []byte("Deterministic test data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute manifest twice
	opts := DefaultChunkOptions()
	manifest1, err := ComputeManifest(testFile, opts)
	if err != nil {
		t.Fatalf("First ComputeManifest failed: %v", err)
	}

	manifest2, err := ComputeManifest(testFile, opts)
	if err != nil {
		t.Fatalf("Second ComputeManifest failed: %v", err)
	}

	// Verify hashes are identical
	if manifest1.Chunks[0].Hash != manifest2.Chunks[0].Hash {
		t.Error("Chunk hashes should be identical for same file")
	}
	if manifest1.MerkleRoot != manifest2.MerkleRoot {
		t.Error("Merkle roots should be identical for same file")
	}
}

func TestReadChunk(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chunks.bin")
	
	chunkSize := 1024
	testData := make([]byte, chunkSize*3)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read first chunk
	chunk0, err := ReadChunk(testFile, 0, chunkSize)
	if err != nil {
		t.Fatalf("ReadChunk(0) failed: %v", err)
	}
	if len(chunk0) != chunkSize {
		t.Errorf("Expected chunk size %d, got %d", chunkSize, len(chunk0))
	}

	// Read second chunk
	chunk1, err := ReadChunk(testFile, 1, chunkSize)
	if err != nil {
		t.Fatalf("ReadChunk(1) failed: %v", err)
	}
	if len(chunk1) != chunkSize {
		t.Errorf("Expected chunk size %d, got %d", chunkSize, len(chunk1))
	}

	// Verify chunks are different
	// Note: chunks may have same pattern due to byte(i % 256) repeating
	// Just verify they read correctly from different positions
	if len(chunk0) == 0 || len(chunk1) == 0 {
		t.Error("Chunks should not be empty")
	}

	// Verify chunk data matches original
	for i := 0; i < chunkSize; i++ {
		if chunk0[i] != testData[i] {
			t.Errorf("Chunk 0 byte %d mismatch", i)
			break
		}
		if chunk1[i] != testData[chunkSize+i] {
			t.Errorf("Chunk 1 byte %d mismatch", i)
			break
		}
	}
}

func TestComputeManifest_EmptyFile(t *testing.T) {
	// Create empty file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.bin")
	
	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute manifest
	opts := DefaultChunkOptions()
	manifest, err := ComputeManifest(testFile, opts)
	if err != nil {
		t.Fatalf("ComputeManifest failed: %v", err)
	}

	// Verify empty file handling
	if manifest.FileSize != 0 {
		t.Errorf("Expected file size 0, got %d", manifest.FileSize)
	}
	if manifest.ChunkCount != 1 {
		t.Errorf("Expected 1 chunk for empty file, got %d", manifest.ChunkCount)
	}
}

func TestComputeManifest_FileNotFound(t *testing.T) {
	// Try to compute manifest for non-existent file
	_, err := ComputeManifest("/nonexistent/file.bin", DefaultChunkOptions())
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
