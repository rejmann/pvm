//go:build windows

package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/php"
)

func WindowsInstall(h *home.Dir, ver string) error {
	branch := majorMinor(ver)

	fullVer, err := resolveFullVersion(ver, branch)
	if err != nil {
		return fmt.Errorf("resolve PHP %s: %w", ver, err)
	}

	installDir := h.PHPDir(branch)
	fmt.Printf("Downloading PHP %s to %s...\n", fullVer, installDir)

	if err := downloadAndExtractPHP(fullVer, installDir); err != nil {
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

func WindowsRemove(h *home.Dir, ver string) error {
	branch := majorMinor(ver)
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
	return php.LatestPatch(context.Background(), branch)
}

// WindowsEnsureExtensions enables exts in the php.ini of an installed version.
func WindowsEnsureExtensions(h *home.Dir, ver string, exts []string) error {
	return enableIniExtensions(h.PHPDir(majorMinor(ver)), exts)
}
