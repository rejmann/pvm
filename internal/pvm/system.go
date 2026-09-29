package pvm

import (
	"errors"

	"github.com/rejmann/pvm/internal/home"
)

// current returns the global version, adopting the PHP installed before pvm
// when there is none yet.
func (m *Manager) current() (string, error) {
	v, err := m.Home.Current()
	if errors.Is(err, home.ErrNoCurrentVersion) && m.adoptSystem() {
		return m.Home.Current()
	}
	return v, err
}

// adoptSystem runs once per pvm home: if no global version is set, the php
// the user already runs (installed before pvm) is recorded as an installed
// version and made the global one, so it is in use without pvm use. pvm
// never uninstalls it. It reports whether a version was adopted.
func (m *Manager) adoptSystem() bool {
	if m.SystemPHP == nil || m.Home.SystemChecked() {
		return false
	}
	if err := m.Home.SetSystemChecked(); err != nil {
		return false
	}
	if _, err := m.Home.Current(); !errors.Is(err, home.ErrNoCurrentVersion) {
		return false
	}

	php, ok := m.SystemPHP()
	if !ok {
		return false
	}
	v, installed := m.Home.Match(php.Version)
	if !installed {
		v = php.Version
		if err := m.Home.SetSystem(v, php.Binary); err != nil {
			return false
		}
	}
	return m.Home.SetCurrent(v) == nil
}
