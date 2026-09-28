//go:build windows

package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rejmann/pvm/internal/catalog"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/version"
)

// Install downloads PHP ver from windows.php.net into the pvm home and
// returns its php.exe.
func Install(ctx context.Context, h *home.Dir, ver string, s proc.Streams) (string, error) {
	branch := version.Branch(ver)

	fullVer, err := resolveFullVersion(ctx, ver, branch)
	if err != nil {
		return "", fmt.Errorf("resolve PHP %s: %w", ver, err)
	}

	installDir := h.PHPDir(branch)
	fmt.Fprintf(s.Out, "Downloading PHP %s to %s...\n", fullVer, installDir)

	if err := downloadAndExtractPHP(ctx, fullVer, installDir, s); err != nil {
		return "", err
	}

	bin := filepath.Join(installDir, "php.exe")
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("php.exe not found at %s after extraction", bin)
	}
	if err := writePHPIni(installDir); err != nil {
		return "", fmt.Errorf("write php.ini: %w", err)
	}
	return bin, nil
}

func Remove(h *home.Dir, ver string, s proc.Streams) error {
	installDir := h.PHPDir(version.Branch(ver))

	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed", ver)
	}

	return os.RemoveAll(installDir)
}

// resolveFullVersion returns the full patch version.
// If ver already has a patch (e.g. "8.3.30"), it is returned as-is.
// Otherwise (e.g. "8.3"), the latest patch is fetched from php.net.
func resolveFullVersion(ctx context.Context, ver, branch string) (string, error) {
	if v, err := version.Parse(ver); err == nil && v.HasPatch() {
		return ver, nil
	}
	return catalog.LatestPatch(ctx, branch)
}

// EnsureExtensions writes php.ini for an installed version that has none (e.g.
// one installed before pvm wrote php.ini). An existing php.ini is the user's:
// pvm only says what to add.
func EnsureExtensions(h *home.Dir, ver string, s proc.Streams) error {
	installDir := h.PHPDir(version.Branch(ver))
	iniPath := filepath.Join(installDir, "php.ini")
	if _, err := os.Stat(iniPath); err == nil {
		return fmt.Errorf("%s already exists — add this to it:\n%s", iniPath, pvmIniBlock(installDir))
	}
	return writePHPIni(installDir)
}
