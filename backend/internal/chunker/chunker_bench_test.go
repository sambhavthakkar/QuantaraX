package chunker

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func BenchmarkChunker(b *testing.B) {
	buf := make([]byte, 8<<20)
	rand.Read(buf)
	bm := bytes.NewReader(buf)
	for n := 0; n < b.N; n++ {
		c, _ := NewChunker(bm, 64<<10)
		for {
			_, err := c.Next()
			if err != nil { break }
		}
		bm.Seek(0, 0)
	}
}
