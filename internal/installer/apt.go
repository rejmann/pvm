package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func AptInstall(base, ver string) error {
	branch := majorMinor(ver)
	pkg := "php" + branch + "-cli"

	if err := aptInstallPkg(pkg); err != nil {
		fmt.Println("Package not found, adding ondrej/php PPA...")
		if ppaErr := addOndrejPPA(); ppaErr != nil {
			return fmt.Errorf("add ondrej/php PPA: %w", ppaErr)
		}
		if err2 := aptInstallPkg(pkg); err2 != nil {
			return fmt.Errorf("install PHP %s: %w", ver, err2)
		}
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

func AptRemove(base, ver string) error {
	branch := majorMinor(ver)
	pkg := "php" + branch + "-cli"

	cmd := exec.Command("sudo", "apt-get", "remove", "-y", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-get remove %s: %w", pkg, err)
	}
	return nil
}

func aptInstallPkg(pkg string) error {
	cmd := exec.Command("sudo", "apt-get", "install", "-y", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func addOndrejPPA() error {
	add := exec.Command("sudo", "add-apt-repository", "-y", "ppa:ondrej/php")
	add.Stdout = os.Stdout
	add.Stderr = os.Stderr
	if err := add.Run(); err != nil {
		return err
	}
	update := exec.Command("sudo", "apt-get", "update")
	update.Stdout = os.Stdout
	update.Stderr = os.Stderr
	return update.Run()
}

func majorMinor(ver string) string {
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ver
}
