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

func SetCurrent(base, version, binaryPath string) error {
	switch runtime.GOOS {
	case system.Linux:
		return setCurrentLinux(base, version, binaryPath)
	case system.Windows:
		return setCurrentWindows(base, version, binaryPath)
	default:
		return setCurrentShim(base, version, binaryPath)
	}
}

func RemoveCurrent(base string) error {
	switch runtime.GOOS {
	case system.Linux:
		return removeCurrentLinux(base)
	default:
		return removeCurrentShim(base)
	}
}

func setCurrentLinux(base, version, binaryPath string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", binaryPath, err)
	}
	return writeCurrentVersion(base, version)
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

func removeCurrentShim(base string) error {
	shimPath := filepath.Join(base, "shims", "php")
	if err := os.Remove(shimPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove php shim: %w", err)
	}
	return removeCurrentVersion(base)
}

func setCurrentWindows(base, version, binaryPath string) error {
	shimDir := filepath.Join(base, "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("create shims directory: %w", err)
	}

	content := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", binaryPath)
	shimPath := filepath.Join(shimDir, "php.bat")
	if err := os.WriteFile(shimPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write php shim: %w", err)
	}

	return writeCurrentVersion(base, version)
}

func writeCurrentVersion(base, version string) error {
	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(version), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

func removeCurrentVersion(base string) error {
	vf := filepath.Join(base, "current-version")
	if err := os.Remove(vf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
