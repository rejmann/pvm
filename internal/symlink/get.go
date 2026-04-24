package symlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoCurrentVersion = errors.New("no current version set")

func GetCurrent(base string) (string, error) {
	data, err := os.ReadFile(filepath.Join(base, "current-version"))
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
