package chunker

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zeebo/blake3"
)

// ComputeManifest generates a complete manifest for the given file
func ComputeManifest(filePath string, options ChunkOptions) (*Manifest, error) {
	// Validate chunk size
	if options.ChunkSize <= 0 {
		options = DefaultChunkOptions()
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := fileInfo.Size()
	fileName := filepath.Base(filePath)

	// Calculate chunk count
	chunkCount := int(fileSize) / options.ChunkSize
	if int(fileSize)%options.ChunkSize != 0 {
		chunkCount++
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Handle empty files
	if fileSize == 0 {
		// For empty files, create one empty chunk
		hasher := blake3.New()
		hash := hasher.Sum(nil)
		hashBase64 := base64.StdEncoding.EncodeToString(hash)
		
		chunks := []ChunkDescriptor{{
			Index:  0,
			Hash:   hashBase64,
			Length: 0,
		}}
		
		merkleRoot, _ := ComputeMerkleRoot([]string{hashBase64})
		
		return &Manifest{
			SessionID:  sessionID,
			FileName:   fileName,
			FileSize:   0,
			ChunkSize:  options.ChunkSize,
			ChunkCount: 1,
			HashAlgo:   "BLAKE3",
			Chunks:     chunks,
			MerkleRoot: merkleRoot,
			CreatedAt:  time.Now(),
		}, nil
	}

	// Compute chunk hashes
	chunks := make([]ChunkDescriptor, 0, chunkCount)
	chunkHashes := make([]string, 0, chunkCount)
	buffer := make([]byte, options.ChunkSize)

	for i := 0; ; i++ {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		if n == 0 {
			break
		}

		// Compute BLAKE3 hash
		hasher := blake3.New()
		hasher.Write(buffer[:n])
		hash := hasher.Sum(nil)
		hashBase64 := base64.StdEncoding.EncodeToString(hash)

		// Add chunk descriptor
		chunks = append(chunks, ChunkDescriptor{
			Index:  i,
			Hash:   hashBase64,
			Length: n,
		})
		chunkHashes = append(chunkHashes, hashBase64)

		if err == io.EOF {
			break
		}
	}

	// Compute Merkle root
	merkleRoot, err := ComputeMerkleRoot(chunkHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to compute merkle root: %w", err)
	}

	// Create manifest
	manifest := &Manifest{
		SessionID:  sessionID,
		FileName:   fileName,
		FileSize:   fileSize,
		ChunkSize:  options.ChunkSize,
		ChunkCount: len(chunks),
		HashAlgo:   "BLAKE3",
		Chunks:     chunks,
		MerkleRoot: merkleRoot,
		CreatedAt:  time.Now(),
	}

	return manifest, nil
}

// Chunker provides streaming chunking of data from an io.Reader
type Chunker struct {
	reader    io.Reader
	chunkSize int
	buffer    []byte
}

// NewChunker creates a new streaming chunker
func NewChunker(r io.Reader, chunkSize int) (*Chunker, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be positive")
	}
	return &Chunker{
		reader:    r,
		chunkSize: chunkSize,
		buffer:    make([]byte, chunkSize),
	}, nil
}

// Next returns the next chunk of data
func (c *Chunker) Next() ([]byte, error) {
	n, err := c.reader.Read(c.buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n == 0 {
		return nil, io.EOF
	}
	return c.buffer[:n], nil
}

// ReadChunk reads a specific chunk from the file
func ReadChunk(filePath string, chunkIndex int, chunkSize int) ([]byte, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Calculate offset
	offset := int64(chunkIndex) * int64(chunkSize)

	// Seek to offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	// Read chunk
	buffer := make([]byte, chunkSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	return buffer[:n], nil
}
