package symlink

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func SetCurrent(base, version, binaryPath string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--set", "php", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-alternatives --set php %s: %w", binaryPath, err)
	}

	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(version), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

func RemoveCurrent(base string) error {
	cmd := exec.Command("sudo", "update-alternatives", "--auto", "php")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("update-alternatives --auto php: %w", err)
	}

	vf := filepath.Join(base, "current-version")
	if err := os.Remove(vf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
