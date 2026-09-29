package pvm

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/sysphp"
)

// ExtensionManager adds, removes and toggles the PHP extensions of installed
// versions.
type ExtensionManager interface {
	ExtensionInstaller
	RemoveExtensions(h *home.Dir, ver string, exts []string) (phpext.Removal, error)
	SetExtensionsEnabled(h *home.Dir, ver string, exts []string, enabled bool) error
}

// Extensions manages the PHP extensions of the versions pvm installed.
type Extensions struct {
	Manager   *Manager
	Installer ExtensionManager
	// Probe inspects a php binary (sysphp.Probe).
	Probe func(bin string) (sysphp.Info, error)
}

// Version picks the version to work on, as Select does. A version installed
// outside pvm is refused unless readOnly: pvm never changes it.
func (e *Extensions) Version(arg, dir, env string, readOnly bool) (Active, error) {
	a, err := e.Manager.Select(arg, dir, env)
	if err != nil {
		return Active{}, err
	}
	if a.System && !readOnly {
		return Active{}, fmt.Errorf("PHP %s was installed outside pvm, so pvm does not change its extensions — manage them with the tool that installed it", a.Version)
	}
	return a, nil
}

// List returns the extensions a's PHP loads, sorted.
func (e *Extensions) List(a Active) ([]string, error) {
	info, err := e.Probe(a.Binary)
	if err != nil {
		return nil, err
	}
	return info.Extensions, nil
}

// Add installs the extensions of exts a's PHP does not load yet. It returns
// the ones it installed and the ones already loaded, named as by phpext.Name.
func (e *Extensions) Add(a Active, exts []string) (added, loaded []string, err error) {
	info, err := e.Probe(a.Binary)
	if err != nil {
		return nil, nil, err
	}
	for _, ext := range names(exts) {
		if info.Has(ext) {
			loaded = append(loaded, ext)
		} else {
			added = append(added, ext)
		}
	}
	if len(added) == 0 {
		return nil, loaded, nil
	}
	if err := e.Installer.AddExtensions(e.Manager.Home, a.Version, added); err != nil {
		return nil, loaded, err
	}
	return added, loaded, nil
}

// Remove uninstalls the extensions of exts pvm installed from a's PHP and
// disables the ones that ship with it. The ones it can do neither with fail
// the call, after the others were handled: compiled into PHP, or not there.
func (e *Extensions) Remove(a Active, exts []string) (phpext.Removal, error) {
	r, err := e.Installer.RemoveExtensions(e.Manager.Home, a.Version, names(exts))
	if err != nil || len(r.Stuck) == 0 {
		return r, err
	}

	info, err := e.Probe(a.Binary)
	if err != nil {
		return r, err
	}
	var builtin, absent []string
	for _, ext := range r.Stuck {
		if info.Has(ext) {
			builtin = append(builtin, ext)
		} else {
			absent = append(absent, ext)
		}
	}
	var errs []error
	if len(builtin) > 0 {
		errs = append(errs, fmt.Errorf("%s %s compiled into PHP %s: it cannot be removed or disabled",
			strings.Join(builtin, ", "), isAre(builtin), a.Version))
	}
	if len(absent) > 0 {
		errs = append(errs, fmt.Errorf("PHP %s has no %s extension", a.Version, strings.Join(absent, ", ")))
	}
	return r, errors.Join(errs...)
}

func isAre(list []string) string {
	if len(list) == 1 {
		return "is"
	}
	return "are"
}

// SetEnabled turns exts on or off for a's PHP without uninstalling them.
func (e *Extensions) SetEnabled(a Active, exts []string, enabled bool) error {
	return e.Installer.SetExtensionsEnabled(e.Manager.Home, a.Version, names(exts), enabled)
}

// names spells exts as PHP does, without blanks or repeats, in order.
func names(exts []string) []string {
	var out []string
	for _, ext := range exts {
		if n := phpext.Name(ext); n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}
