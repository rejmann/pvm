package pvm

import (
	"fmt"

	"github.com/rejmann/pvm/internal/version"
)

// Target is a version the user asked for, with the "lts" alias resolved.
type Target struct {
	Arg     string // as typed, e.g. "lts"
	Version string // concrete version, e.g. "8.4"
	Alias   bool   // Arg was an alias
}

// String labels the version for messages: "8.4", or "8.4 (lts)" for the alias.
func (t Target) String() string {
	if t.Alias {
		return t.Version + " (lts)"
	}
	return t.Version
}

// Target resolves arg (a version or "lts") into a valid concrete version.
func (m *Manager) Target(arg string) (Target, error) {
	concrete, alias, err := version.Resolve(arg, m.LTS)
	if err != nil {
		return Target{}, err
	}
	if _, err := version.Parse(concrete); err != nil {
		return Target{}, fmt.Errorf("invalid version %q: %w", concrete, err)
	}
	return Target{Arg: arg, Version: concrete, Alias: alias}, nil
}
