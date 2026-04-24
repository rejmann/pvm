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

	if _, err := exec.LookPath("winget"); err == nil {
		if err := wingetInstall(ver, branch); err == nil {
			return windowsSaveBinary(base, ver, branch)
		}
		fmt.Printf("winget could not install PHP %s, trying Chocolatey...\n", ver)
	}

	if _, err := exec.LookPath("choco"); err != nil {
		return fmt.Errorf(
			"no package manager found\n\n" +
				"Install via winget:  winget install PHP.PHP." + branch + "\n" +
				"Or install Chocolatey from https://chocolatey.org",
		)
	}

	if err := chocoInstall(ver); err != nil {
		return err
	}
	return windowsSaveBinary(base, ver, branch)
}

func WindowsRemove(base, ver string) error {
	branch := majorMinor(ver)

	if _, err := exec.LookPath("winget"); err == nil {
		cmd := exec.Command("winget", "uninstall", "--id", "PHP.PHP."+branch, "--silent")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if _, err := exec.LookPath("choco"); err != nil {
		return fmt.Errorf("no package manager found to remove PHP %s", ver)
	}

	cmd := exec.Command("choco", "uninstall", "php", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("choco uninstall php: %w", err)
	}
	return nil
}

func wingetInstall(ver, branch string) error {
	pkgID := "PHP.PHP." + branch
	cmd := exec.Command(
		"winget", "install",
		"--id", pkgID,
		"--silent",
		"--accept-package-agreements",
		"--accept-source-agreements",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("winget install %s: %w", pkgID, err)
	}
	return nil
}

func chocoInstall(ver string) error {
	cmd := exec.Command("choco", "install", "php", "--version="+ver, "-y", "--allow-downgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("choco install php --version=%s: %w", ver, err)
	}
	return nil
}

func windowsSaveBinary(base, ver, branch string) error {
	binPath, err := findWindowsBinary(branch)
	if err != nil {
		return err
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func findWindowsBinary(branch string) (string, error) {
	compact := strings.ReplaceAll(branch, ".", "")
	candidates := []string{
		// winget (official PHP Windows installer)
		filepath.Join("C:\\", "php", "php.exe"),
		filepath.Join("C:\\", "php"+branch, "php.exe"),
		filepath.Join("C:\\", "Program Files", "PHP", "php-"+branch, "php.exe"),
		filepath.Join("C:\\", "Program Files", "PHP", "v"+branch, "php.exe"),
		// chocolatey
		filepath.Join("C:\\", "tools", "php"+compact, "php.exe"),
		filepath.Join("C:\\", "tools", "php", "php.exe"),
		filepath.Join("C:\\", "ProgramData", "chocolatey", "lib", "php", "tools", "php.exe"),
		filepath.Join("C:\\", "php"+compact, "php.exe"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if p, err := exec.LookPath("php"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("PHP binary not found after installation — check your PHP install location")
}
