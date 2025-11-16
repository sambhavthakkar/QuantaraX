package scenarios

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantarax/backend/internal/media"
	"github.com/quantarax/backend/internal/chunker"
)

// Test_MoovRelocation_CorruptedTail creates a minimal MP4-like file with moov at tail,
// calls RelocateMoovToFront, and verifies moov precedes mdat afterward without panic.
func Test_MoovRelocation_CorruptedTail(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "tailmoov.mp4")
	// Build minimal boxes: [ftyp][mdat][moov]
	ftyp := make([]byte, 24)
	binary.BigEndian.PutUint32(ftyp[0:4], uint32(len(ftyp)))
	copy(ftyp[4:8], []byte("ftyp"))
	copy(ftyp[8:12], []byte("isom"))
	mdat := make([]byte, 16)
	binary.BigEndian.PutUint32(mdat[0:4], uint32(len(mdat)))
	copy(mdat[4:8], []byte("mdat"))
	moov := make([]byte, 24)
	binary.BigEndian.PutUint32(moov[0:4], uint32(len(moov)))
	copy(moov[4:8], []byte("moov"))
	// append fake stco atom inside moov with zero entries
	binary.BigEndian.PutUint32(moov[8:12], uint32(8))
	copy(moov[12:16], []byte("stco"))
	binary.BigEndian.PutUint32(moov[16:20], uint32(0)) // version/flags placeholder
	binary.BigEndian.PutUint32(moov[20:24], uint32(0)) // entry_count 0
	if err := os.WriteFile(p, append(append(ftyp, mdat...), moov...), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := media.RelocateMoovToFront(p); err != nil {
		t.Fatalf("relocate err: %v", err)
	}
	out, err := os.ReadFile(p)
	if err != nil { t.Fatalf("read: %v", err) }
	// Scan order of atoms to ensure moov before mdat
	find := func(tag string) int {
		for i := 0; i+8 <= len(out); {
			sz := int(binary.BigEndian.Uint32(out[i : i+4]))
			if sz < 8 || i+sz > len(out) { break }
			if string(out[i+4:i+8]) == tag { return i }
			i += sz
		}
		return -1
	}
	moovOff := find("moov")
	mdatOff := find("mdat")
	if moovOff == -1 || mdatOff == -1 { t.Fatalf("atoms not found after relocation") }
	if !(moovOff < mdatOff) { t.Fatalf("expected moov before mdat, got moovOff=%d mdatOff=%d", moovOff, mdatOff) }
}

// Test_CAS_Skip_Scheduling verifies that CAS bitmap detection produces expected ranges for skip.
func Test_CAS_Skip_Scheduling(t *testing.T) {
	// Build a dummy manifest with 5 chunks and hashes from data
	dir := t.TempDir()
	file := filepath.Join(dir, "data.bin")
	payload := make([]byte, 5*256)
	for i := range payload { payload[i] = byte(i%251) }
	if err := os.WriteFile(file, payload, 0644); err != nil { t.Fatalf("write: %v", err) }
	mf, err := chunker.ComputeManifest(file, chunker.ChunkOptions{ChunkSize: 256})
	if err != nil { t.Fatalf("manifest: %v", err) }
	// Init CAS and pre-populate with first two chunk hashes
	tc := &testCAS{m: map[string]bool{}}
	for _, ch := range mf.Chunks[:2] { tc.m[ch.Hash] = true }
	// Build ranges using same logic as receiver
	var idxs []int64
	for _, ch := range mf.Chunks { if tc.HasChunk(ch.Hash) { idxs = append(idxs, int64(ch.Index)) } }
	// Local simple range compressor (mirrors transport behavior)
	type rc struct{}
	compress := func(idxs []int64) string {
		if len(idxs) == 0 { return "" }
		r := ""
		s := idxs[0]
		p := idxs[0]
		for i := 1; i < len(idxs); i++ {
			c := idxs[i]
			if c == p+1 { p = c; continue }
			if s == p { r += fmt.Sprintf("%d,", s) } else { r += fmt.Sprintf("%d-%d,", s, p) }
			s = c; p = c
		}
		if s == p { r += fmt.Sprintf("%d", s) } else { r += fmt.Sprintf("%d-%d", s, p) }
		return r
	}
	ranges := compress(idxs)
	if ranges != "0-1" {
		t.Fatalf("expected ranges '0-1', got %q", ranges)
	}
}

type testCAS struct{ m map[string]bool }
func (t *testCAS) HasChunk(hash string) bool { return t.m[hash] }
func (t *testCAS) PutChunk(hash string, length int) error { t.m[hash] = true; return nil }

