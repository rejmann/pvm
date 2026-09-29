package pvm

import (
	"errors"
	"fmt"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/project"
)

// EnvVersion overrides every other source of the active version.
const EnvVersion = "PVM_VERSION"

var ErrNoActiveVersion = errors.New("no PHP version selected")

// Active is the PHP version selected for a directory and where it came from.
type Active struct {
	Version string // installed version, e.g. "8.3"
	Binary  string // path to the php binary
	Source  string // human-readable origin: env var, .php-version path or "global"
	System  bool   // installed outside pvm and adopted by it
}

// Global reports whether the version comes from pvm use rather than a
// project file or $PVM_VERSION.
func (a Active) Global() bool {
	return a.Source == sourceGlobal
}

const sourceGlobal = "global"

// Active picks the PHP version for dir, in order of precedence: env (the
// value of $PVM_VERSION), the nearest .php-version, then the global version.
func (m *Manager) Active(dir, env string) (Active, error) {
	requested, source, err := m.requested(dir, env)
	if err != nil {
		return Active{}, err
	}

	installed, ok := m.Home.Match(requested)
	if !ok {
		return Active{}, fmt.Errorf("PHP %s (set by %s) is not installed — run: pvm install %s",
			requested, source, requested)
	}

	bin, err := m.Home.Binary(installed)
	if err != nil {
		return Active{}, err
	}
	return Active{Version: installed, Binary: bin, Source: source, System: m.Home.System(installed)}, nil
}

func (m *Manager) requested(dir, env string) (v, source string, err error) {
	if env != "" {
		return env, EnvVersion + " environment variable", nil
	}

	v, path, err := project.Find(dir)
	if err == nil {
		return v, path, nil
	}
	if !errors.Is(err, project.ErrNotFound) {
		return "", "", err
	}

	v, err = m.current()
	if err == nil {
		return v, sourceGlobal, nil
	}
	if errors.Is(err, home.ErrNoCurrentVersion) {
		return "", "", ErrNoActiveVersion
	}
	return "", "", err
}

// Lookup resolves arg (a version, branch or "lts") to an installed version
// and its php binary.
func (m *Manager) Lookup(arg string) (Active, error) {
	t, err := m.Target(arg)
	if err != nil {
		return Active{}, err
	}

	installed, ok := m.Home.Match(t.Version)
	if !ok {
		return Active{}, fmt.Errorf("PHP %s is not installed — run: pvm install %s", t.Version, t.Version)
	}
	bin, err := m.Home.Binary(installed)
	if err != nil {
		return Active{}, err
	}
	if bin == "" {
		return Active{}, errors.New("empty binary path for PHP " + installed)
	}
	return Active{Version: installed, Binary: bin, Source: "argument", System: m.Home.System(installed)}, nil
}

// Select returns the version to run: arg when given, otherwise the version
// in use for dir (see Active).
func (m *Manager) Select(arg, dir, env string) (Active, error) {
	if arg != "" {
		return m.Lookup(arg)
	}
	return m.Active(dir, env)
}
