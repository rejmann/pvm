// Package resolve decides which installed PHP runs: the one selected for a
// directory (Active) or one named explicitly (Version).
package resolve

import (
	"errors"
	"fmt"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/project"
	"github.com/rejmann/pvm/internal/version"
)

// EnvVersion overrides every other source of the active version.
const EnvVersion = "PVM_VERSION"

var ErrNoActiveVersion = errors.New("no PHP version selected")

// Target is an installed PHP version and where it was selected.
type Target struct {
	Version string // installed version, e.g. "8.3"
	Binary  string // path to the php binary
	Source  string // human-readable origin: env var, .php-version path or "global"
}

// Active picks the PHP version for dir, in order of precedence: env (the
// value of $PVM_VERSION), the nearest .php-version, then the global version.
func Active(h *home.Dir, dir, env string) (Target, error) {
	requested, source, err := requestedVersion(h, dir, env)
	if err != nil {
		return Target{}, err
	}

	installed, ok := h.MatchInstalled(requested)
	if !ok {
		return Target{}, fmt.Errorf("PHP %s (set by %s) is not installed — run: pvm install %s",
			requested, source, requested)
	}

	bin, err := h.VersionBinary(installed)
	if err != nil {
		return Target{}, err
	}

	return Target{Version: installed, Binary: bin, Source: source}, nil
}

func requestedVersion(h *home.Dir, dir, env string) (v, source string, err error) {
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

	v, err = h.Current()
	if err == nil {
		return v, "global", nil
	}
	if errors.Is(err, home.ErrNoCurrentVersion) {
		return "", "", ErrNoActiveVersion
	}
	return "", "", err
}

// Version resolves arg (a version, branch or "lts") to an installed version
// and its php binary.
func Version(h *home.Dir, arg string, r version.Resolver) (Target, error) {
	concrete, _, err := version.Resolve(arg, r)
	if err != nil {
		return Target{}, err
	}

	if _, err := version.Parse(concrete); err != nil {
		return Target{}, fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	installed, ok := h.MatchInstalled(concrete)
	if !ok {
		return Target{}, fmt.Errorf("PHP %s is not installed — run: pvm install %s", concrete, concrete)
	}

	bin, err := h.VersionBinary(installed)
	if err != nil {
		return Target{}, err
	}
	if bin == "" {
		return Target{}, errors.New("empty binary path for PHP " + installed)
	}
	return Target{Version: installed, Binary: bin, Source: "argument"}, nil
}
