//go:build linux || darwin

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func BrewInstall(base, ver string) error {
	branch := majorMinor(ver)
	pkg := "php@" + branch

	cmd := exec.Command("brew", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install %s: %w\n\nMake sure Homebrew is installed: https://brew.sh", pkg, err)
	}

	binPath, err := brewBinaryPath(branch)
	if err != nil {
		return err
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func BrewRemove(base, ver string) error {
	branch := majorMinor(ver)
	cmd := exec.Command("brew", "uninstall", "php@"+branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew uninstall php@%s: %w", branch, err)
	}
	return nil
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
