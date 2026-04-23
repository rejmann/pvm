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

func (m *Manager) EnsureBaseDir() error {
	return os.MkdirAll(m.versionsDir(), 0755)
}
