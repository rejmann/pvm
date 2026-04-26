//go:build linux || darwin

package symlink

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rejmann/pvm/system"
)

func RemoveCurrent(base string) error {
	switch runtime.GOOS {
	case system.Linux:
		return removeCurrentLinux(base)
	default:
		return removeCurrentShim(base)
	}
}

func SetCurrent(base, version, binaryPath string) error {
	switch runtime.GOOS {
	case system.Linux:
		return setCurrentLinux(base, version, binaryPath)
	default:
		return setCurrentShim(base, version, binaryPath)
	}
}

func setCurrentLinux(base, version, binaryPath string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", binaryPath, err)
	}

	if err := updateSymlink(filepath.Join(base, "bin", "php"), binaryPath); err != nil {
		return fmt.Errorf("update pvm bin symlink: %w", err)
	}

	localBin := filepath.Join(filepath.Dir(base), ".local", "bin", "php")
	if fi, err := os.Lstat(localBin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := updateSymlink(localBin, binaryPath); err != nil {
			return fmt.Errorf("update ~/.local/bin/php symlink: %w", err)
		}
	}

	return writeCurrentVersion(base, version)
}

func setCurrentShim(base, version, binaryPath string) error {
	shimDir := filepath.Join(base, "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("create shims directory: %w", err)
	}

	shimPath := filepath.Join(shimDir, "php")
	_ = os.Remove(shimPath)
	if err := os.Symlink(binaryPath, shimPath); err != nil {
		return fmt.Errorf("create php shim: %w", err)
	}

	return writeCurrentVersion(base, version)
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

func removeCurrentLinux(base string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--auto", "php")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}
	return removeCurrentVersion(base)
}

func removeCurrentShim(base string) error {
	shimPath := filepath.Join(base, "shims", "php")
	if err := os.Remove(shimPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove php shim: %w", err)
	}
	return removeCurrentVersion(base)
}

func removeCurrentVersion(base string) error {
	vf := filepath.Join(base, "current-version")
	if err := os.Remove(vf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
