//go:build darwin

package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
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

// Remove runs brew uninstall php@X.Y.
func (s *System) Remove(h *home.Dir, ver string) error {
	branch := version.Branch(ver)
	cmd := exec.Command("brew", "uninstall", "php@"+branch)
	cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew uninstall php@%s: %w", branch, err)
	}
	return nil
}

// AddExtensions is not supported: Homebrew's php@X.Y bundles the common
// extensions, and PECL ones are not managed yet.
func (s *System) AddExtensions(_ *home.Dir, _ string, exts []string) error {
	return fmt.Errorf("pvm cannot install PHP extensions with Homebrew yet (%s)", strings.Join(exts, ", "))
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
