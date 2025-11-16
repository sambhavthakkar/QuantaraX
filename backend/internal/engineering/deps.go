package engineering

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverDependencies scans for simple BOM/manifest files listing dependencies.
// Placeholder: reads lines as file paths and returns fully qualified paths that exist.
func DiscoverDependencies(root string) ([]string, error) {
	var deps []string
	entries, err := os.ReadDir(root)
	if err != nil { return nil, err }
	for _, e := range entries {
		if e.IsDir() { continue }
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, "bom.txt") || strings.HasSuffix(name, "manifest.txt") {
			p := filepath.Join(root, e.Name())
			f, err := os.Open(p)
			if err != nil { continue }
			s := bufio.NewScanner(f)
			for s.Scan() {
				cand := filepath.Join(root, s.Text())
				if _, err := os.Stat(cand); err == nil {
					deps = append(deps, cand)
				}
			}
			f.Close()
		}
	}
	return deps, nil
}
