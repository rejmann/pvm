//go:build linux || darwin

package selfupdate

import (
	"errors"
	"os"
	"path/filepath"
)

// RemoveBinary deletes exe. A running pvm keeps its own inode, so it can
// delete itself.
func RemoveBinary(exe string) error {
	if err := os.Remove(exe); err != nil && !errors.Is(err, os.ErrNotExist) {
		return wrapRemove(err, filepath.Dir(exe))
	}
	return nil
}
