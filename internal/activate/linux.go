//go:build linux

package activate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
)

// Use makes version (whose php binary is bin) the global PHP: /usr/bin/php
// follows it through update-alternatives, for services that skip the shim.
func Use(h *home.Dir, version, bin string, s proc.Streams) error {
	if err := s.Run("sudo", "update-alternatives", "--set", "php", bin); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", bin, err)
	}

	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}

	localBin := filepath.Join(filepath.Dir(h.Base), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, bin); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
	}

	return h.SetCurrent(version)
}

// Clear leaves no global version: update-alternatives goes back to automatic
// mode and the shim falls back to the system php.
func Clear(h *home.Dir, s proc.Streams) error {
	if err := s.Run("sudo", "update-alternatives", "--auto", "php"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}
	return h.ClearCurrent()
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
