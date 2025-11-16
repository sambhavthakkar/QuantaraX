package engineering

import (
	"io"
	"os"

	"github.com/zeebo/blake3"
)

// ComputeDeltaBlocks computes rolling hashes for fixed-size blocks to allow delta-sync.
// This is a simple baseline: fixed window; production may use variable windows and sparse maps.
func ComputeDeltaBlocks(path string, blockSize int) ([][32]byte, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var hashes [][32]byte
	buf := make([]byte, blockSize)
	for {
		n, err := io.ReadFull(f, buf)
		if err == io.ErrUnexpectedEOF {
			// last partial block
			h := blake3.Sum256(buf[:n])
			hashes = append(hashes, h)
			break
		}
		if err == io.EOF { break }
		if err != nil { return nil, err }
		h := blake3.Sum256(buf)
		hashes = append(hashes, h)
	}
	return hashes, nil
}
