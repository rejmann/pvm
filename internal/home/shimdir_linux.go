package home

import "path/filepath"

// ShimDir holds the php shim; users add it to their PATH.
func (d *Dir) ShimDir() string {
	return filepath.Join(d.Path, "bin")
}
