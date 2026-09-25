package cmd

import (
	"errors"
	"fmt"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/project"
	"github.com/rejmann/pvm/internal/symlink"
)

// envVersion overrides every other source of the active version.
const envVersion = "PVM_VERSION"

var ErrNoActiveVersion = errors.New("no PHP version selected")

// active is the PHP version selected for a directory and where it came from.
type active struct {
	Version string // installed version, e.g. "8.3"
	Binary  string // path to the php binary
	Source  string // human-readable origin: env var, .php-version path or "global"
}

// resolveActive picks the PHP version for dir, in order of precedence:
// $PVM_VERSION, the nearest .php-version, then the global current-version.
func resolveActive(m *phpfs.Manager, dir, env string) (active, error) {
	requested, source, err := requestedVersion(m, dir, env)
	if err != nil {
		return active{}, err
	}

	installed, ok := m.MatchInstalled(requested)
	if !ok {
		return active{}, fmt.Errorf("PHP %s (set by %s) is not installed — run: pvm install %s",
			requested, source, requested)
	}

	bin, err := m.GetVersionBinary(installed)
	if err != nil {
		return active{}, err
	}

	return active{Version: installed, Binary: bin, Source: source}, nil
}

func requestedVersion(m *phpfs.Manager, dir, env string) (version, source string, err error) {
	if env != "" {
		return env, envVersion + " environment variable", nil
	}

	v, path, err := project.Find(dir)
	if err == nil {
		return v, path, nil
	}
	if !errors.Is(err, project.ErrNotFound) {
		return "", "", err
	}

	v, err = symlink.GetCurrent(m.Base)
	if err == nil {
		return v, "global", nil
	}
	if errors.Is(err, symlink.ErrNoCurrentVersion) {
		return "", "", ErrNoActiveVersion
	}
	return "", "", err
}
