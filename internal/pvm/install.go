package pvm

import (
	"context"
	"fmt"
)

// Install installs t unless it is already installed. onStart, if not nil, is
// called once the checks pass, right before the (slow) installation, which
// stops when ctx is cancelled.
func (m *Manager) Install(ctx context.Context, t Target, onStart func()) error {
	if err := m.Home.Init(); err != nil {
		return fmt.Errorf("initialize \"pvm\" directory: %w", err)
	}
	m.adoptSystem()
	if m.Home.Installed(t.Version) {
		return fmt.Errorf("%s already installed", t)
	}

	if onStart != nil {
		onStart()
	}
	if err := m.Installer.Install(ctx, m.Home, t.Version); err != nil {
		return fmt.Errorf("install PHP %s: %w", t.Version, err)
	}
	return nil
}
