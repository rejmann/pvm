package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func WindowsInstall(base, ver string) error {
	branch := majorMinor(ver)

	if _, err := exec.LookPath("winget"); err != nil {
		return fmt.Errorf("winget not found — install App Installer from the Microsoft Store")
	}

	pkgID, err := wingetFindPHP(branch)
	if err != nil {
		return err
	}

	// each branch gets its own isolated directory under pvm home
	installDir := phpInstallDir(base, branch)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}

	cmd := exec.Command(
		"winget", "install",
		"--id", pkgID,
		"--location", installDir,
		"--silent",
		"--accept-package-agreements",
		"--accept-source-agreements",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("winget install %s: %w", pkgID, err)
	}

	binPath := filepath.Join(installDir, "php.exe")
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func WindowsRemove(base, ver string) error {
	branch := majorMinor(ver)

	if _, err := exec.LookPath("winget"); err != nil {
		return fmt.Errorf("winget not found — install App Installer from the Microsoft Store")
	}

	pkgID, err := wingetFindPHP(branch)
	if err != nil {
		return fmt.Errorf("PHP %s not found in winget: %w", ver, err)
	}

	installDir := phpInstallDir(base, branch)
	cmd := exec.Command(
		"winget", "uninstall",
		"--id", pkgID,
		"--location", installDir,
		"--silent",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("winget uninstall %s: %w", pkgID, err)
	}
	return nil
}

func wingetFindPHP(branch string) (string, error) {
	out, err := exec.Command("winget", "search", "PHP.PHP", "--source", "winget").Output()
	if err != nil {
		return "", fmt.Errorf("winget search: %w", err)
	}

	target := "PHP.PHP." + branch
	for _, line := range strings.Split(string(out), "\n") {
		for _, f := range strings.Fields(line) {
			if strings.EqualFold(f, target) {
				return target, nil
			}
		}
	}

	return "", fmt.Errorf("PHP %s not found in winget — run: pvm available", branch)
}

// phpInstallDir returns the isolated directory for a PHP branch under pvm home.
// e.g. %LOCALAPPDATA%\pvm\php\8.3
func phpInstallDir(base, branch string) string {
	return filepath.Join(base, "php", branch)
}
