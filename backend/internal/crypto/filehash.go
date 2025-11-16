package crypto

import (
	"encoding/base64"
	"io"
	"os"

	"github.com/zeebo/blake3"
)

// ComputeFileHashB64 computes BLAKE3 of a file and returns base64-encoded digest.
func ComputeFileHashB64(path string) string {
	f, err := os.Open(path)
	if err != nil { return "" }
	defer f.Close()
	h := blake3.New()
	buf := make([]byte, 1<<20)
	for {
		n, err := f.Read(buf)
		if n > 0 { h.Write(buf[:n]) }
		if err == io.EOF { break }
		if err != nil { return "" }
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
