package chunker

import (
	"encoding/base64"

	"github.com/zeebo/blake3"
)

// ComputeMerkleRoot computes the Merkle root from chunk hashes
func ComputeMerkleRoot(chunkHashes []string) (string, error) {
	if len(chunkHashes) == 0 {
		return "", nil
	}

	// Decode base64 hashes to bytes
	hashes := make([][]byte, len(chunkHashes))
	for i, hashStr := range chunkHashes {
		decoded, err := base64.StdEncoding.DecodeString(hashStr)
		if err != nil {
			return "", err
		}
		hashes[i] = decoded
	}

	// Build Merkle tree bottom-up
	for len(hashes) > 1 {
		var nextLevel [][]byte
		
		// Process pairs
		for i := 0; i < len(hashes); i += 2 {
			var combined []byte
			
			if i+1 < len(hashes) {
				// Pair exists: hash(left || right)
				combined = append(hashes[i], hashes[i+1]...)
			} else {
				// Odd element: duplicate it
				combined = append(hashes[i], hashes[i]...)
			}
			
			// Hash the combined pair
			hasher := blake3.New()
			hasher.Write(combined)
			parentHash := hasher.Sum(nil)
			nextLevel = append(nextLevel, parentHash)
		}
		
		hashes = nextLevel
	}

	// Encode root as base64
	return base64.StdEncoding.EncodeToString(hashes[0]), nil
}
