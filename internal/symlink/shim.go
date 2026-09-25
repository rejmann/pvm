package symlink

import (
	"path/filepath"
	"runtime"

	"github.com/rejmann/pvm/internal/system"
)

// ShimDir is the directory holding the php shim; users add it to their PATH.
func ShimDir(base string) string {
	if runtime.GOOS == system.Linux {
		return filepath.Join(base, "bin")
	}
	return filepath.Join(base, "shims")
}
