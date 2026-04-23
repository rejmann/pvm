package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrVersionNotInstalled = errors.New("version not installed")

func (m *Manager) GetVersionBinary(v string) (string, error) {
	data, err := os.ReadFile(filepath.Join(m.VersionDir(v), "binary"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrVersionNotInstalled, v)
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *Manager) VersionInstalled(v string) bool {
	binPath, err := m.GetVersionBinary(v)
	if err != nil {
		return false
	}
	_, err = os.Stat(binPath)
	return err == nil
}
