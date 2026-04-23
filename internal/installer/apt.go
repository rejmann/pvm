package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Apt installs a PHP version using the system apt package manager (requires sudo).
// The ondrej/php PPA must be configured for non-distro PHP versions.
func Apt(base, ver string) error {
	branch := majorMinor(ver)
	pkg := "php" + branch + "-cli"

	cmd := exec.Command("sudo", "apt-get", "install", "-y", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-get install %s: %w\n\nIf the package was not found, add the ondrej/php PPA first:\n  sudo add-apt-repository ppa:ondrej/php && sudo apt-get update", pkg, err)
	}

	binPath := "/usr/bin/php" + branch
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func majorMinor(ver string) string {
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ver
}
