// Package home owns the layout of the pvm data directory ($PVM_HOME, ~/.pvm
// or %LOCALAPPDATA%\pvm). No other package builds paths inside it:
//
//	current-version          global version, written by pvm use
//	versions/<v>/binary      path of the php binary of installed version v
//	versions/<v>/packages    extension packages pvm installed for v (Linux)
//	versions/<v>/system      v was installed outside pvm: adopted, never uninstalled
//	system-checked           pvm already looked for a PHP installed before it
//	php/<branch>/            PHP builds pvm extracted itself (Windows)
//	bin/ or shims/           the php shim, the directory users add to PATH
//	cache/                   cached php.net data
//	composer/                everything pvm composer keeps
package home

import (
	"os"
	"path/filepath"
)

// Dir is a pvm data directory.
type Dir struct {
	Path string
}

// New returns the pvm data directory at path.
func New(path string) *Dir {
	return &Dir{Path: path}
}

// Default returns the pvm data directory in use: $PVM_HOME, or the
// platform's default location.
func Default() *Dir {
	if v := os.Getenv("PVM_HOME"); v != "" {
		return New(v)
	}
	return New(defaultPath())
}

func (d *Dir) versionsDir() string {
	return filepath.Join(d.Path, "versions")
}

// VersionDir holds the metadata of installed version v.
func (d *Dir) VersionDir(v string) string {
	return filepath.Join(d.versionsDir(), v)
}

// PHPDir is where pvm extracts the PHP build of branch (e.g. "8.3") itself.
func (d *Dir) PHPDir(branch string) string {
	return filepath.Join(d.PHPRoot(), branch)
}

// PHPRoot holds every PHP build pvm extracted itself.
func (d *Dir) PHPRoot() string {
	return filepath.Join(d.Path, "php")
}

// AvailableCache is the cached list of PHP branches from php.net.
func (d *Dir) AvailableCache() string {
	return filepath.Join(d.Path, "cache", "available.json")
}

// ComposerDir holds the Composer of every PHP version and its shared cache.
func (d *Dir) ComposerDir() string {
	return filepath.Join(d.Path, "composer")
}

// Init creates the directories every pvm command expects.
func (d *Dir) Init() error {
	return os.MkdirAll(d.versionsDir(), 0755)
}
