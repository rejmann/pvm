// Package home owns the layout of the pvm home directory. Every path under
// it is built here, so no other package knows where pvm keeps its state.
//
//	current-version        global version, written by pvm use
//	versions/<v>/binary    path of the php binary of each installed version
//	bin/ or shims/         the php shim (ShimDir), which users add to PATH
//	php/<branch>/          PHP builds pvm downloads itself (Windows)
//	composer/              per-PHP Composer, see package composer
//	cache/                 cached php.net data
package home

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dir is a pvm home directory.
type Dir struct {
	Base string
}

func New(base string) *Dir {
	return &Dir{Base: base}
}

// Default is the pvm home of the current user: $PVM_HOME, or the platform
// default (defaultBase).
func Default() *Dir {
	if v := os.Getenv("PVM_HOME"); v != "" {
		return New(v)
	}
	return New(defaultBase())
}

func (d *Dir) EnsureBaseDir() error {
	return os.MkdirAll(d.versionsDir(), 0755)
}

// ShimDir holds the php shim; users add it to their PATH.
func (d *Dir) ShimDir() string {
	if runtime.GOOS == "linux" {
		return filepath.Join(d.Base, "bin")
	}
	return filepath.Join(d.Base, "shims")
}

// PHPDir is where pvm unpacks the PHP builds it downloads itself, one
// directory per branch (e.g. php/8.3).
func (d *Dir) PHPDir(branch string) string {
	return filepath.Join(d.Base, "php", branch)
}

// PHPRoot is the parent of every PHPDir.
func (d *Dir) PHPRoot() string {
	return filepath.Join(d.Base, "php")
}

// ComposerDir holds the Composer of every PHP version and their shared cache.
func (d *Dir) ComposerDir() string {
	return filepath.Join(d.Base, "composer")
}

// CacheDir holds data pvm can fetch again, such as the php.net release list.
func (d *Dir) CacheDir() string {
	return filepath.Join(d.Base, "cache")
}
