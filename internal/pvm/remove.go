package pvm

import (
	"fmt"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/version"
)

// Removal reports what Remove did besides removing the version.
type Removal struct {
	WasCurrent  bool  // it was the global version; none is active now
	System      bool  // installed outside pvm: only forgotten, not uninstalled
	ComposerErr error // its Composer could not be deleted
}

// Remove uninstalls version v with its metadata and Composer. A version
// installed outside pvm is only forgotten: pvm never uninstalls it.
func (m *Manager) Remove(v string) (Removal, error) {
	if _, err := version.Parse(v); err != nil {
		return Removal{}, fmt.Errorf("invalid version %q: %w", v, err)
	}
	m.adoptSystem()
	if !m.Home.Installed(v) {
		return Removal{}, fmt.Errorf("PHP %s is not installed", v)
	}

	current, _ := m.Home.Current()
	r := Removal{WasCurrent: current == v, System: m.Home.System(v)}

	if !r.System {
		if err := m.Installer.Remove(m.Home, v); err != nil {
			return Removal{}, fmt.Errorf("remove PHP %s: %w", v, err)
		}
	}
	if err := m.Home.RemoveVersion(v); err != nil {
		return Removal{}, fmt.Errorf("remove PHP %s metadata: %w", v, err)
	}
	r.ComposerErr = composer.Remove(m.Home.ComposerDir(), v)

	if r.WasCurrent {
		_ = m.Activator.Deactivate(m.Home)
	}
	return r, nil
}
