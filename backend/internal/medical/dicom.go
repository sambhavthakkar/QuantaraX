package medical

import (
	"os"
)

// Minimal DICOM detector and placeholder metadata extractor.
// In production, integrate a proper DICOM library.

type Study struct {
	SeriesCount int
}

type Metadata struct {
	Studies []Study
}

func DetectAndExtract(path string) (*Metadata, bool) {
	// Only detect by magic for now
	f, err := os.Open(path)
	if err != nil { return nil, false }
	defer f.Close()
	buf := make([]byte, 132)
	n, _ := f.Read(buf)
	if n < 132 { return nil, false }
	if string(buf[128:132]) != "DICM" { return nil, false }
	// Placeholder metadata
	return &Metadata{Studies: []Study{{SeriesCount: 1}}}, true
}
