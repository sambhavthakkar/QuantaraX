package transport

import (
    "testing"
)

// Fuzz for control stream range compression/decompression symmetry
func FuzzChunkRangeCompression(f *testing.F) {
    seeds := []string{"", "1", "1-3", "1,3-5,7", "0-0,1,2-2", "10,11,12-14,20"}
    for _, s := range seeds { f.Add(s) }
    var c ChunkRangeCompressor
    f.Fuzz(func(t *testing.T, s string) {
        idxs, err := c.Decompress(s)
        if err != nil { return }
        back := c.Compress(idxs)
        idxs2, err2 := c.Decompress(back)
        if err2 != nil { t.Fatalf("roundtrip failed: %v", err2) }
        if len(idxs) != len(idxs2) { t.Fatalf("length mismatch: %d vs %d", len(idxs), len(idxs2)) }
        for i := range idxs { if idxs[i] != idxs2[i] { t.Fatalf("elem mismatch at %d", i) } }
    })
}
