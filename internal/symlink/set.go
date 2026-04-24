package symlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func SetCurrent(base, version, binaryPath string) error {
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}

	// Atomic symlink replacement: write to a temp name, then rename.
	link := filepath.Join(binDir, "php")
	tmp := link + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(binaryPath, tmp); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace symlink: %w", err)
	}

	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(version), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

func RemoveCurrent(base string) error {
	link := filepath.Join(base, "bin", "php")
	if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove symlink: %w", err)
	}
	vf := filepath.Join(base, "current-version")
	if err := os.Remove(vf); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
