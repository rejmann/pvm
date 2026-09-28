//go:build darwin

package activate

import (
	"fmt"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
)

// Use makes version the global PHP; the shim picks it up on the next call.
func Use(h *home.Dir, version, bin string, s proc.Streams) error {
	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}
	return h.SetCurrent(version)
}

// Clear leaves no global version. The shim stays in place: it still serves
// projects that have a .php-version, and otherwise falls back to the system
// php.
func Clear(h *home.Dir, s proc.Streams) error {
	return h.ClearCurrent()
}
