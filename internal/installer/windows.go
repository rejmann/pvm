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

	if _, err := exec.LookPath("choco"); err != nil {
		return fmt.Errorf("Chocolatey not found\n\nInstall Chocolatey from https://chocolatey.org then run this command again")
	}

	cmd := exec.Command("choco", "install", "php", "--version="+ver, "-y", "--allow-downgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("choco install php --version=%s: %w", ver, err)
	}

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

func WindowsRemove(base, ver string) error {
	if _, err := exec.LookPath("choco"); err != nil {
		return fmt.Errorf("Chocolatey not found — cannot remove PHP automatically")
	}

	cmd := exec.Command("choco", "uninstall", "php", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("choco uninstall php: %w", err)
	}
	return nil
}

func findWindowsBinary(branch string) (string, error) {
	compact := strings.ReplaceAll(branch, ".", "")
	candidates := []string{
		filepath.Join("C:\\", "tools", "php"+compact, "php.exe"),
		filepath.Join("C:\\", "tools", "php", "php.exe"),
		filepath.Join("C:\\", "ProgramData", "chocolatey", "lib", "php", "tools", "php.exe"),
		filepath.Join("C:\\", "php"+compact, "php.exe"),
		filepath.Join("C:\\", "php", "php.exe"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if p, err := exec.LookPath("php"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("PHP binary not found after installation — check your Chocolatey PHP install location")
}
