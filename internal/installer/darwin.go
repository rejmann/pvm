//go:build darwin

package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/version"
)

// Install installs PHP ver with Homebrew and returns its php binary.
func Install(ctx context.Context, h *home.Dir, ver string, s proc.Streams) (string, error) {
	pkg := "php@" + version.Branch(ver)

	if err := s.Run("brew", "install", pkg); err != nil {
		return "", fmt.Errorf("brew install %s: %w\n\nMake sure Homebrew is installed: https://brew.sh", pkg, err)
	}

	return brewBinaryPath(pkg)
}

func Remove(h *home.Dir, ver string, s proc.Streams) error {
	pkg := "php@" + version.Branch(ver)
	if err := s.Run("brew", "uninstall", pkg); err != nil {
		return fmt.Errorf("brew uninstall %s: %w", pkg, err)
	}
	return nil
}

// EnsureExtensions is a no-op: Homebrew's php@X.Y already includes zip.
func EnsureExtensions(h *home.Dir, ver string, s proc.Streams) error {
	return nil
}

func brewBinaryPath(pkg string) (string, error) {
	out, err := exec.Command("brew", "--prefix", pkg).Output()
	if err != nil {
		return "", fmt.Errorf("brew --prefix %s: %w", pkg, err)
	}
	prefix := strings.TrimSpace(string(out))
	bin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("PHP binary not found at %s after installation", bin)
	}
	return bin, nil
}
