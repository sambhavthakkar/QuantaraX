package helpers

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/zeebo/blake3"
)

// FileGenerator helps generate test files with known properties
type FileGenerator struct {
	TempDir string
}

// NewFileGenerator creates a new file generator
func NewFileGenerator() (*FileGenerator, error) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("quantarax-test-%d", os.Getpid()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	
	return &FileGenerator{
		TempDir: tempDir,
	}, nil
}

// GenerateFile creates a file with random data of specified size
func (fg *FileGenerator) GenerateFile(name string, size int64) (string, string, error) {
	filePath := filepath.Join(fg.TempDir, name)
	
	file, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Generate random data and compute hash simultaneously
	hasher := blake3.New()
	remaining := size
	bufSize := int64(1024 * 1024) // 1MB buffer
	
	for remaining > 0 {
		toWrite := bufSize
		if remaining < bufSize {
			toWrite = remaining
		}
		
		buf := make([]byte, toWrite)
		if _, err := rand.Read(buf); err != nil {
			return "", "", fmt.Errorf("failed to generate random data: %w", err)
		}
		
		if _, err := file.Write(buf); err != nil {
			return "", "", fmt.Errorf("failed to write data: %w", err)
		}
		
		if _, err := hasher.Write(buf); err != nil {
			return "", "", fmt.Errorf("failed to hash data: %w", err)
		}
		
		remaining -= toWrite
	}
	
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return filePath, hash, nil
}

// GenerateSmallFile creates a small test file (1 MB)
func (fg *FileGenerator) GenerateSmallFile(name string) (string, string, error) {
	return fg.GenerateFile(name, 1*1024*1024)
}

// GenerateMediumFile creates a medium test file (100 MB)
func (fg *FileGenerator) GenerateMediumFile(name string) (string, string, error) {
	return fg.GenerateFile(name, 100*1024*1024)
}

// GenerateLargeFile creates a large test file (1 GB)
func (fg *FileGenerator) GenerateLargeFile(name string) (string, string, error) {
	return fg.GenerateFile(name, 1*1024*1024*1024)
}

// ComputeHash computes BLAKE3 hash of a file
func (fg *FileGenerator) ComputeHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	hasher := blake3.New()
	buf := make([]byte, 1024*1024)
	
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if _, err := hasher.Write(buf[:n]); err != nil {
				return "", fmt.Errorf("failed to hash data: %w", err)
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}
	
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// MakeTempDir creates a subdirectory within the temp directory
func (fg *FileGenerator) MakeTempDir(name string) string {
	dir := filepath.Join(fg.TempDir, name)
	os.MkdirAll(dir, 0755)
	return dir
}

// Cleanup removes all generated files
func (fg *FileGenerator) Cleanup() error {
	return os.RemoveAll(fg.TempDir)
}
