package fs

import "path/filepath"

type Manager struct {
	Base string
}

func NewManager(base string) *Manager {
	return &Manager{Base: base}
}

func (m *Manager) VersionDir(v string) string {
	versionsDir := filepath.Join(m.Base, "versions")

	return filepath.Join(versionsDir, v)
}
