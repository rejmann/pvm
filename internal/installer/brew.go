//go:build darwin

package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/version"
)

// System installs PHP with Homebrew (php@X.Y).
type System struct {
	Stdout, Stderr io.Writer // where brew's output goes
}

// New returns the installer for this system.
func New(stdout, stderr io.Writer) *System {
	return &System{Stdout: stdout, Stderr: stderr}
}

// Install runs brew install php@X.Y and records its binary.
func (s *System) Install(h *home.Dir, ver string) error {
	branch := version.Branch(ver)
	pkg := "php@" + branch

	cmd := exec.Command("brew", "install", pkg)
	cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install %s: %w\n\nMake sure Homebrew is installed: https://brew.sh", pkg, err)
	}

	binPath, err := brewBinaryPath(branch)
	if err != nil {
		return err
	}

	return h.SetBinary(ver, binPath)
}

// Remove runs brew uninstall php@X.Y, after the extension formulas pvm
// installed for it.
func (s *System) Remove(h *home.Dir, ver string) error {
	branch := version.Branch(ver)
	for _, formula := range h.Packages(ver) {
		exec.Command("brew", "uninstall", formula).Run()
	}
	cmd := exec.Command("brew", "uninstall", "php@"+branch)
	cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew uninstall php@%s: %w", branch, err)
	}
	return nil
}

// extTap has a formula per extension and PHP branch (redis@8.3), built for
// the php@X.Y of homebrew-core. Homebrew taps it on the first install.
const extTap = "shivammathur/extensions/"

// AddExtensions installs the formulas of exts for installed version ver and
// records them, so Remove uninstalls them with the version. Homebrew's
// php@X.Y already bundles the core extensions (intl, zip...); the tap adds
// the PECL ones (redis, xdebug, imagick...).
func (s *System) AddExtensions(h *home.Dir, ver string, exts []string) error {
	branch := version.Branch(ver)

	var installed, failed []string
	for _, ext := range exts {
		formula := extTap + phpext.Name(ext) + "@" + branch
		if slices.Contains(installed, formula) || slices.Contains(failed, formula) {
			continue
		}
		cmd := exec.Command("brew", "install", formula)
		cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
		if err := cmd.Run(); err != nil {
			failed = append(failed, formula)
			continue
		}
		installed = append(installed, formula)
	}

	if err := h.AddPackages(ver, installed); err != nil {
		return fmt.Errorf("record installed extensions: %w", err)
	}
	if len(failed) > 0 {
		return fmt.Errorf("could not install %s with Homebrew", strings.Join(failed, ", "))
	}
	return nil
}

// RemoveExtensions uninstalls the formulas pvm installed for exts. The others
// ship with php@X.Y: the ones a conf.d file loads (e.g. opcache) are disabled
// instead, the ones compiled in cannot be removed.
func (s *System) RemoveExtensions(h *home.Dir, ver string, exts []string) (phpext.Removal, error) {
	var r phpext.Removal
	branch := version.Branch(ver)
	owned := h.Packages(ver)
	bin, err := h.Binary(ver)
	if err != nil {
		return r, err
	}

	var formulas, toDisable []string
	for _, ext := range exts {
		formula := extTap + phpext.Name(ext) + "@" + branch
		switch {
		case slices.Contains(owned, formula):
			r.Uninstalled = append(r.Uninstalled, ext)
			if !slices.Contains(formulas, formula) {
				formulas = append(formulas, formula)
			}
		case confdLoads(bin, ext):
			toDisable = append(toDisable, ext)
		default:
			r.Stuck = append(r.Stuck, ext)
		}
	}

	for _, formula := range formulas {
		cmd := exec.Command("brew", "uninstall", formula)
		cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
		if err := cmd.Run(); err != nil {
			return phpext.Removal{}, fmt.Errorf("brew uninstall %s: %w", formula, err)
		}
		if err := h.RemovePackages(ver, []string{formula}); err != nil {
			return phpext.Removal{}, fmt.Errorf("record removed extensions: %w", err)
		}
	}
	if len(toDisable) > 0 {
		if err := setConfdEnabled(bin, toDisable, false, s.Stderr); err != nil {
			return phpext.Removal{Uninstalled: r.Uninstalled}, err
		}
		r.Disabled = toDisable
	}
	return r, nil
}

// SetExtensionsEnabled turns exts on or off by renaming their file in the
// conf.d directory of Homebrew's php@X.Y.
func (s *System) SetExtensionsEnabled(h *home.Dir, ver string, exts []string, enabled bool) error {
	bin, err := h.Binary(ver)
	if err != nil {
		return err
	}
	return setConfdEnabled(bin, exts, enabled, s.Stderr)
}

func brewBinaryPath(branch string) (string, error) {
	out, err := exec.Command("brew", "--prefix", "php@"+branch).Output()
	if err != nil {
		return "", fmt.Errorf("brew --prefix php@%s: %w", branch, err)
	}
	prefix := strings.TrimSpace(string(out))
	binPath := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}
	return binPath, nil
}
