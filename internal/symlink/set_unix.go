//go:build linux || darwin

package symlink

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/system"
)

func RemoveCurrent(h *home.Dir) error {
	switch runtime.GOOS {
	case system.Linux:
		return removeCurrentLinux(h)
	default:
		// The shim stays: with no global version it still serves projects
		// that have a .php-version, and otherwise falls back to the system php.
		return h.ClearCurrent()
	}
}

func SetCurrent(h *home.Dir, version, binaryPath string) error {
	switch runtime.GOOS {
	case system.Linux:
		return setCurrentLinux(h, version, binaryPath)
	default:
		return setCurrentShim(h, version)
	}
}

func setCurrentLinux(h *home.Dir, version, binaryPath string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", binaryPath, err)
	}

	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}

	localBin := filepath.Join(filepath.Dir(h.Path), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, binaryPath); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
	}

	return h.SetCurrent(version)
}

func setCurrentShim(h *home.Dir, version string) error {
	if err := EnsureShim(h); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}

	return h.SetCurrent(version)
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

func removeCurrentLinux(h *home.Dir) error {
	cmd := exec.Command("sudo", "update-alternatives", "--auto", "php")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}
	return h.ClearCurrent()
}

// RemoveIntegration is a no-op on Unix: pvm never edits shell config files,
// and everything else it creates lives in the pvm home.
func RemoveIntegration(h *home.Dir, binDir string) error {
	return nil
}
