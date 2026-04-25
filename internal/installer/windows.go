package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/php"
)

func WindowsInstall(base, ver string) error {
	branch := majorMinor(ver)

	fullVer, err := resolveFullVersion(ver, branch)
	if err != nil {
		return fmt.Errorf("resolve PHP %s: %w", ver, err)
	}

	installDir := phpInstallDir(base, branch)
	fmt.Printf("Downloading PHP %s to %s...\n", fullVer, installDir)

	if err := downloadAndExtractPHP(fullVer, branch, installDir); err != nil {
		return err
	}

	binPath := filepath.Join(installDir, "php.exe")
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("php.exe not found at %s after extraction", binPath)
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func WindowsRemove(base, ver string) error {
	branch := majorMinor(ver)
	installDir := phpInstallDir(base, branch)

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
	return php.LatestPatch(context.Background(), branch)
}

// phpInstallDir is the isolated directory for a PHP branch under pvm home.
// e.g. %LOCALAPPDATA%\pvm\php\8.3
func phpInstallDir(base, branch string) string {
	return filepath.Join(base, "php", branch)
}
