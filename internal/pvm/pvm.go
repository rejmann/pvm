// Package pvm implements what pvm does — installing, selecting, resolving and
// removing PHP versions — independently of the command line. Everything that
// touches the system comes in through the small interfaces below, so the
// rules here are tested without installing anything.
package pvm

import (
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/sysphp"
	"github.com/rejmann/pvm/internal/version"
)

// Installer installs and removes PHP versions on this system.
type Installer interface {
	Install(h *home.Dir, ver string) error
	Remove(h *home.Dir, ver string) error
}

// Activator makes a version the global one, or leaves none active.
type Activator interface {
	Activate(h *home.Dir, ver, bin string) error
	Deactivate(h *home.Dir) error
}

// Manager runs pvm's use cases against one pvm home.
type Manager struct {
	Home      *home.Dir
	Installer Installer
	Activator Activator
	LTS       version.Resolver    // resolves the "lts" alias
	System    func() []sysphp.PHP // PHP installed outside pvm
	// SystemPHP finds the php already in use outside pvm, adopted as the
	// global version when pvm has none (see adoptSystem).
	SystemPHP func() (sysphp.PHP, bool)
}
