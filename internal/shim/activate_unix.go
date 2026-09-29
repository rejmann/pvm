//go:build !windows

package shim

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rejmann/pvm/internal/home"
)

// Activator switches the global version with the shim. It never needs root:
// /usr/bin/php is left to the system, and programs that do not go through
// PATH (services, cron) should name the version's binary (pvm which).
type Activator struct {
	Stdout, Stderr io.Writer
}

// New returns the activator for this system.
func New(stdout, stderr io.Writer) *Activator {
	return &Activator{Stdout: stdout, Stderr: stderr}
}

// Activate makes ver, whose php binary is bin, the global version.
func (a *Activator) Activate(h *home.Dir, ver, bin string) error {
	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}
	// Before pvm used bin/ everywhere, the macOS shim lived in shims/; a
	// stale one first on PATH would keep running the old pvm binary.
	os.RemoveAll(filepath.Join(h.Path, "shims"))

	localBin := filepath.Join(filepath.Dir(h.Path), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, bin); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
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

func updateSymlink(link, target string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return err
	}
	tmp := link + ".tmp"
	os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, link)
}
