package pvm

import (
	"errors"
	"fmt"

	"github.com/rejmann/pvm/internal/process"
)

// Shim returns the php binary the php shim runs in dir: the version in use
// (see Active) or, when none is selected anywhere, the first php on path
// outside pvm's shim directory.
func (m *Manager) Shim(dir, env, path string) (string, error) {
	a, err := m.Active(dir, env)
	if err == nil {
		return a.Binary, nil
	}
	if !errors.Is(err, ErrNoActiveVersion) {
		return "", err
	}

	if bin := process.LookPath("php", path, m.Home.ShimDir()); bin != "" {
		return bin, nil
	}
	return "", fmt.Errorf("%w and no system php found — run: pvm use <version>", err)
}
