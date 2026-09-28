package home

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoCurrentVersion = errors.New("no current version set")

// CurrentFile holds the global version as plain text ("8.3").
func (d *Dir) CurrentFile() string {
	return filepath.Join(d.Base, "current-version")
}

// Current is the global version set by pvm use.
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

func (d *Dir) SetCurrent(v string) error {
	if err := os.WriteFile(d.CurrentFile(), []byte(v), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}

// ClearCurrent unsets the global version; clearing it twice is not an error.
func (d *Dir) ClearCurrent() error {
	if err := os.Remove(d.CurrentFile()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove current-version: %w", err)
	}
	return nil
}
