package home

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoCurrentVersion = errors.New("no current version set")

// CurrentFile holds the global version; the Windows shim reads it directly.
func (d *Dir) CurrentFile() string {
	return filepath.Join(d.Path, "current-version")
}

// Current returns the global version set by pvm use.
func (d *Dir) Current() (string, error) {
	data, err := os.ReadFile(d.CurrentFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoCurrentVersion
		}
		return "", fmt.Errorf("read current-version: %w", err)
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", ErrNoCurrentVersion
	}
	return v, nil
}

// SetCurrent makes v the global version.
func (d *Dir) SetCurrent(v string) error {
	if err := os.MkdirAll(d.Path, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(d.CurrentFile(), []byte(v), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

// ClearCurrent leaves no global version; clearing twice is fine.
func (d *Dir) ClearCurrent() error {
	if err := os.Remove(d.CurrentFile()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
