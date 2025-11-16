package fec

import (
	"crypto/rand"
	"testing"
)

func BenchmarkFECEncode(b *testing.B) {
	data := make([]byte, 1<<20)
	rand.Read(data)
	for i := 0; i < b.N; i++ {
		_ = len(data) // placeholder until FEC encode exposed
	}
}
