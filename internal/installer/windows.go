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

	if err := wingetInstall(ver, branch); err != nil {
		return err
	}
	return windowsSaveBinary(base, ver, branch)
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

	cmd := exec.Command("winget", "uninstall", "--id", pkgID, "--silent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("winget uninstall %s: %w", pkgID, err)
	}
	return nil
}

func wingetInstall(ver, branch string) error {
	pkgID, err := wingetFindPHP(branch)
	if err != nil {
		return err
	}

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

func wingetFindPHP(branch string) (string, error) {
	out, err := exec.Command("winget", "search", "PHP.PHP", "--source", "winget").Output()
	if err != nil {
		return "", fmt.Errorf("winget search: %w", err)
	}

	target := "PHP.PHP." + branch
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.EqualFold(f, target) {
				return target, nil
			}
		}
	}

	return "", fmt.Errorf("PHP %s not found in winget", branch)
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
