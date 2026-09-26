//go:build linux || darwin

package selfupdate

import "os"

// swap renames the new binary over exe. The rename gives the new binary its own
// inode, so a running pvm keeps its old copy and macOS never sees a signed
// binary modified in place (which it would kill on launch).
func swap(newBin, exe string) error {
	return os.Rename(newBin, exe)
}

// RemoveOld is a no-op on Unix: swap leaves nothing behind.
func RemoveOld(exe string) {}
