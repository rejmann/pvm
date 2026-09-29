//go:build !windows

package home

import "path/filepath"

// ShimDir holds the php shim and the pvm binary itself; users add it to
// their PATH. It is theirs, so neither needs root.
func (d *Dir) ShimDir() string {
	return filepath.Join(d.Path, "bin")
}
