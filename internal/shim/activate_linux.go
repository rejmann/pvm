package shim

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rejmann/pvm/internal/home"
)

// Activator switches the global version with the shim and update-alternatives.
type Activator struct {
	Stdout, Stderr io.Writer // where update-alternatives' output goes
}

// New returns the activator for this system.
func New(stdout, stderr io.Writer) *Activator {
	return &Activator{Stdout: stdout, Stderr: stderr}
}

// Activate makes ver, whose php binary is bin, the global version.
func (a *Activator) Activate(h *home.Dir, ver, bin string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", bin)
	cmd.Stdout, cmd.Stderr = a.Stdout, a.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", bin, err)
	}

	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}

	localBin := filepath.Join(filepath.Dir(h.Path), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, bin); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
	}

	return h.SetCurrent(ver)
}

// Deactivate leaves no global version and lets update-alternatives pick php.
func (a *Activator) Deactivate(h *home.Dir) error {
	cmd := exec.Command("sudo", "update-alternatives", "--auto", "php")
	cmd.Stdout, cmd.Stderr = a.Stdout, a.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}
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
