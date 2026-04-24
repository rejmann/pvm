package fs

import (
	"os"
	"path/filepath"
)

type Manager struct {
	Base string
}

func NewManager(base string) *Manager {
	return &Manager{Base: base}
}

func (m *Manager) versionsDir() string {
	return filepath.Join(m.Base, "versions")
}

func (m *Manager) VersionDir(v string) string {
	return filepath.Join(m.versionsDir(), v)
}

func (m *Manager) EnsurebaseDir() error {
	return os.MkdirAll(m.versionsDir(), 0755)
}

func (m *Manager) RemoveVersionDir(v string) error {
	return os.RemoveAll(m.VersionDir(v))
}
