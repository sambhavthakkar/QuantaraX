package media

import (
	"encoding/binary"
	"os"
)

// RelocateMoovToFront moves moov before mdat and rewrites stco/co64 offsets.
// Returns the same path after atomic swap, or empty string if no change needed.
func RelocateMoovToFront(path string) (string, error) {
	pos := DetectMoovPosition(path)
	if pos != "tail" { return "", nil }
	in, err := os.ReadFile(path)
	if err != nil { return "", err }
	// Find moov and mdat boxes with simple scan
	find := func(tag string) (off int, size int) {
		for i := 0; i+8 <= len(in); {
			sz := int(binary.BigEndian.Uint32(in[i:i+4]))
			if sz < 8 || i+sz > len(in) { break }
			if string(in[i+4:i+8]) == tag { return i, sz }
			i += sz
		}
		return -1, 0
	}
	moovOff, moovSize := find("moov")
	mdatOff, _ := find("mdat")
	if moovOff < 0 || mdatOff < 0 { return "", nil }
	if moovOff < mdatOff { return "", nil }
	// Adjust stco/co64 offsets inside moov by delta = size of ftyp (first box) moved ahead of moov
	delta := int64(moovSize)
	moov := make([]byte, moovSize)
	copy(moov, in[moovOff:moovOff+moovSize])
	// Walk atoms inside moov to find 'stco' and 'co64'
	for j := 8; j+8 <= len(moov); {
		sz := int(binary.BigEndian.Uint32(moov[j:j+4]))
		if sz < 8 || j+sz > len(moov) { break }
		typ := string(moov[j+4:j+8])
		if typ == "stco" {
			// version(1) flags(3) entry_count(4) entries(4*count)
			if j+12 > len(moov) { break }
			count := int(binary.BigEndian.Uint32(moov[j+8+4 : j+8+8]))
			off := j + 8 + 8
			for k := 0; k < count && off+4 <= len(moov); k++ {
				v := int64(binary.BigEndian.Uint32(moov[off:off+4]))
				v += delta
				binary.BigEndian.PutUint32(moov[off:off+4], uint32(v))
				off += 4
			}
		} else if typ == "co64" {
			if j+16 > len(moov) { break }
			count := int(binary.BigEndian.Uint32(moov[j+8+4 : j+8+8]))
			off := j + 8 + 8
			for k := 0; k < count && off+8 <= len(moov); k++ {
				v := int64(binary.BigEndian.Uint64(moov[off:off+8]))
				v += delta
				binary.BigEndian.PutUint64(moov[off:off+8], uint64(v))
				off += 8
			}
		}
		j += sz
	}
	// Build new file: ftyp (first box) + moov + rest excluding original moov
	ftypSize := int(binary.BigEndian.Uint32(in[0:4]))
	out := make([]byte, 0, len(in))
	out = append(out, in[:ftypSize]...)
	out = append(out, moov...)
	out = append(out, in[ftypSize:moovOff]...)
	out = append(out, in[moovOff+moovSize:]...)
	tmp := path + ".moovtmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil { return "", err }
	if err := os.Rename(tmp, path); err != nil { return "", err }
	return path, nil
}

// DetectMoovPosition inspects atoms and returns "head", "tail", or "unknown".
func DetectMoovPosition(path string) string {
	f, err := os.Open(path)
	if err != nil { return "unknown" }
	defer f.Close()
	type atom struct { typ [4]byte; size uint32; off int64 }
	var atoms []atom
	var offset int64
	buf := make([]byte, 8)
	for i := 0; i < 100000; i++ { // safety cap
		_, err := f.ReadAt(buf, offset)
		if err != nil { break }
		sz := binary.BigEndian.Uint32(buf[0:4])
		if sz < 8 { break }
		copyType := [4]byte{buf[4],buf[5],buf[6],buf[7]}
		atoms = append(atoms, atom{typ: copyType, size: sz, off: offset})
		offset += int64(sz)
	}
	if len(atoms) == 0 { return "unknown" }
	moovIdx := -1
	for i, a := range atoms { if string(a.typ[:]) == "moov" { moovIdx = i } }
	if moovIdx == -1 { return "unknown" }
	if moovIdx == 0 || moovIdx == 1 { return "head" }
	if moovIdx == len(atoms)-1 { return "tail" }
	return "head" // default to head when interleaved
}
