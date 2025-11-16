package introspect

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Domain represents a detected transfer domain
const (
	DomainMedia        = "media"
	DomainRural        = "rural"
	DomainMedical      = "medical"
	DomainEngineering  = "engineering"
	DomainTelemetry    = "racetrack_factory"
	DomainDisaster     = "disaster"
)

// Decision contains the domain and minimal evidence used
type Decision struct {
	Domain   string
	Evidence map[string]string
}

// Decide determines the domain for a given file path deterministically using
// file signatures, extensions, and basic directory patterns. This is a minimal
// implementation; more rules can be added incrementally.
func Decide(inputPath string) Decision {
	abs := inputPath
	// Try basic extension-based routing first (cheap and deterministic)
	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".dcm", ".nii", ".nii.gz", ".mhd", ".raw", ".nrrd", ".hdr":
		if isDICOM(abs) {
			return Decision{Domain: DomainMedical, Evidence: map[string]string{"reason": "dicom"}}
		}
		return Decision{Domain: DomainMedical, Evidence: map[string]string{"reason": "medical_ext"}}
	case ".mov", ".mp4", ".mxf", ".exr", ".dpx", ".r3d", ".ari", ".cine", ".wav", ".aif":
		if hasMP4Ftyp(abs) || hasMXF(abs) || hasEXR(abs) || hasDPX(abs) {
			return Decision{Domain: DomainMedia, Evidence: map[string]string{"reason": "media_magic"}}
		}
		return Decision{Domain: DomainMedia, Evidence: map[string]string{"reason": "media_ext"}}
	case ".step", ".stp", ".iges", ".igs", ".dwg", ".dxf", ".stl", ".obj", ".gltf", ".glb", ".cgns":
		return Decision{Domain: DomainEngineering, Evidence: map[string]string{"reason": "cad_ext"}}
	case ".mdf", ".blf", ".asc", ".csv", ".parquet", ".bin":
		// If paired with nearby mp4/mov files in same dir, treat as telemetry domain
		if hasSiblingVideo(abs) && looksTelemetry(abs) {
			return Decision{Domain: DomainTelemetry, Evidence: map[string]string{"reason": "telemetry+video"}}
		}
	}

	// Telemetry run detection: siblings telemetry/ and video/ directories and sync.json
	dir := filepath.Dir(abs)
	if hasDirs(dir, []string{"telemetry", "video"}) || hasSyncJSON(dir) {
		return Decision{Domain: DomainTelemetry, Evidence: map[string]string{"reason": "telemetry_layout"}}
	}
	// Directory heuristics for media sequences
	base := strings.ToLower(filepath.Base(filepath.Dir(abs)))
	if base == "seq" || base == "plates" || base == "renders" || strings.HasPrefix(base, "shot") {
		return Decision{Domain: DomainMedia, Evidence: map[string]string{"reason": "media_dir"}}
	}

	// Disaster vs Rural: without network signals in this minimal pass, default to rural
	return Decision{Domain: DomainRural, Evidence: map[string]string{"reason": "default_rural"}}
}

func isDICOM(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	// DICOM has "DICM" magic at offset 128
	buf := make([]byte, 132)
	n, _ := ioReadFull(f, buf)
	if n < 132 { return false }
	return string(buf[128:132]) == "DICM"
}

func hasMP4Ftyp(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	buf := make([]byte, 12)
	n, _ := ioReadFull(f, buf)
	if n < 12 { return false }
	// size(4) + type(4) where type might be 'ftyp' at offset 4
	return string(buf[4:8]) == "ftyp"
}

func hasMXF(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	buf := make([]byte, 16)
	n, _ := ioReadFull(f, buf)
	if n < 4 { return false }
	// KLV key starts 06 0E 2B 34
	return buf[0] == 0x06 && buf[1] == 0x0E && buf[2] == 0x2B && buf[3] == 0x34
}

func hasEXR(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	buf := make([]byte, 4)
	n, _ := ioReadFull(f, buf)
	if n < 4 { return false }
	// 0x762F3101 (little endian)
	magic := binary.LittleEndian.Uint32(buf)
	return magic == 0x01312F76
}

func hasDPX(path string) bool {
	f, err := os.Open(path)
	if err != nil { return false }
	defer f.Close()
	buf := make([]byte, 4)
	n, _ := ioReadFull(f, buf)
	if n < 4 { return false }
	return string(buf) == "SDPX" || string(buf) == "XPDS"
}

func hasSiblingVideo(path string) bool {
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil { return false }
	for _, e := range entries {
		if e.IsDir() { continue }
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".mp4" || ext == ".mov" { return true }
	}
	return false
}

func hasDirs(root string, names []string) bool {
	entries, err := os.ReadDir(root)
	if err != nil { return false }
	m := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() { m[strings.ToLower(e.Name())] = true }
	}
	for _, n := range names {
		if !m[strings.ToLower(n)] { return false }
	}
	return true
}

func hasSyncJSON(root string) bool {
	p := filepath.Join(root, "sync.json")
	b, err := os.ReadFile(p)
	if err != nil { return false }
	var tmp map[string]interface{}
	return json.Unmarshal(b, &tmp) == nil
}

func looksTelemetry(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".csv" {
		f, err := os.Open(path)
		if err != nil { return false }
		defer f.Close()
		r := bufio.NewReader(f)
		line, _ := r.ReadString('\n')
		l := strings.ToLower(line)
		return strings.Contains(l, "timestamp") || strings.Contains(l, "can_id") || strings.Contains(l, "rpm")
	}
	// Other binary formats skipped in this minimal pass
	return ext == ".mdf" || ext == ".blf" || ext == ".asc"
}

// Minimal io.ReadFull alternative to avoid importing io for a single symbol
func ioReadFull(f *os.File, b []byte) (int, error) {
	read := 0
	for read < len(b) {
		n, err := f.Read(b[read:])
		if n > 0 { read += n }
		if err != nil { return read, err }
	}
	return read, nil
}
