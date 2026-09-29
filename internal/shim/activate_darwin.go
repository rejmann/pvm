package shim

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/home"
)

// Activator switches the global version with the shim.
type Activator struct {
	Stdout, Stderr io.Writer
}

// New returns the activator for this system.
func New(stdout, stderr io.Writer) *Activator {
	return &Activator{Stdout: stdout, Stderr: stderr}
}

// Activate makes ver the global version.
func (a *Activator) Activate(h *home.Dir, ver, _ string) error {
	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}
	return h.SetCurrent(ver)
}

// Deactivate leaves no global version. The shim stays: it still serves
// projects that have a .php-version, and otherwise falls back to the system php.
func (a *Activator) Deactivate(h *home.Dir) error {
	return h.ClearCurrent()
}

// RemoveIntegration is a no-op: pvm never edits shell config files, and
// everything else it creates lives in the pvm home.
func (a *Activator) RemoveIntegration(*home.Dir, string) error {
	return nil
}
