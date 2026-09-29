package pvm

import (
	"errors"
	"fmt"

	"github.com/rejmann/pvm/internal/project"
)

// Use makes the installed version t the global one.
func (m *Manager) Use(t Target) error {
	if !m.Home.Installed(t.Version) {
		return fmt.Errorf("%s not installed — run: pvm install %s", t, t.Arg)
	}

	bin, err := m.Home.Binary(t.Version)
	if err != nil {
		return err
	}
	if err := m.Activator.Activate(m.Home, t.Version, bin); err != nil {
		return fmt.Errorf("activate PHP %s: %w", t.Version, err)
	}
	return nil
}

// ProjectVersion returns the version declared in the .php-version nearest to
// dir, and that file's path.
func ProjectVersion(dir string) (v, path string, err error) {
	v, path, err = project.Find(dir)
	if errors.Is(err, project.ErrNotFound) {
		return "", "", fmt.Errorf("no version given and no %s found — run: pvm use <version>", project.FileName)
	}
	return v, path, err
}
