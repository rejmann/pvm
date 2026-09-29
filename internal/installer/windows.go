//go:build windows

package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/phpnet"
	"github.com/rejmann/pvm/internal/version"
)

// System installs the PHP builds from windows.php.net into the pvm home.
type System struct {
	Stdout, Stderr io.Writer // progress messages
}

// New returns the installer for this system.
func New(stdout, stderr io.Writer) *System {
	return &System{Stdout: stdout, Stderr: stderr}
}

// Install downloads and extracts PHP ver, writes its php.ini and records
// its binary.
func (s *System) Install(h *home.Dir, ver string) error {
	branch := version.Branch(ver)

	fullVer, err := resolveFullVersion(ver, branch)
	if err != nil {
		return fmt.Errorf("resolve PHP %s: %w", ver, err)
	}

	installDir := h.PHPDir(branch)
	fmt.Fprintf(s.Stdout, "Downloading PHP %s to %s...\n", fullVer, installDir)

	if err := downloadAndExtractPHP(fullVer, installDir, s.Stdout); err != nil {
		return err
	}

	binPath := filepath.Join(installDir, "php.exe")
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("php.exe not found at %s after extraction", binPath)
	}
	if err := writePHPIni(installDir); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}

	return h.SetBinary(ver, binPath)
}

// Remove deletes the extracted build of ver.
func (s *System) Remove(h *home.Dir, ver string) error {
	branch := version.Branch(ver)
	installDir := h.PHPDir(branch)

	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed", ver)
	}

	return os.RemoveAll(installDir)
}

// resolveFullVersion returns the full patch version.
// If ver already has three parts (e.g. "8.3.30"), it is returned as-is.
// Otherwise (e.g. "8.3"), the latest patch is fetched from php.net.
func resolveFullVersion(ver, branch string) (string, error) {
	if len(strings.Split(ver, ".")) == 3 {
		return ver, nil
	}
	return phpnet.LatestPatch(context.Background(), branch)
}

// AddExtensions enables exts in the php.ini of installed version ver.
func (s *System) AddExtensions(h *home.Dir, ver string, exts []string) error {
	return enableIniExtensions(h.PHPDir(version.Branch(ver)), exts)
}

// RemoveExtensions stops the php.ini of installed version ver from loading
// exts. Their DLLs ship with the PHP build, so they are disabled (commented
// out), not deleted; the ones php.ini has no line for are compiled in.
func (s *System) RemoveExtensions(h *home.Dir, ver string, exts []string) (phpext.Removal, error) {
	missing, err := disableIniExtensions(h.PHPDir(version.Branch(ver)), exts)
	if err != nil {
		return phpext.Removal{}, err
	}
	r := phpext.Removal{Stuck: missing}
	for _, ext := range exts {
		if !slices.Contains(missing, phpext.Name(ext)) {
			r.Disabled = append(r.Disabled, ext)
		}
	}
	return r, nil
}

// SetExtensionsEnabled comments exts out of, or back into, the php.ini of
// installed version ver.
func (s *System) SetExtensionsEnabled(h *home.Dir, ver string, exts []string, enabled bool) error {
	dir := h.PHPDir(version.Branch(ver))
	if enabled {
		return enableIniExtensions(dir, exts)
	}
	missing, err := disableIniExtensions(dir, exts)
	if err == nil && len(missing) > 0 {
		err = fmt.Errorf("php.ini does not load %s", strings.Join(missing, ", "))
	}
	return err
}
