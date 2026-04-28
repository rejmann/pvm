//go:build linux || darwin

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	pmApt    = "apt-get"
	pmDnf    = "dnf"
	pmYum    = "yum"
	pmPacman = "pacman"
	pmZypper = "zypper"

	phpBinDir = "/usr/bin/php"
)

type pkgManagerDef struct {
	bin         string
	phpPkg      func(branch string) string
	phpBin      func(branch string) string
	installArgs func(pkg string) []string
	removeArgs  func(pkg string) []string
	preInstall  func(branch string) error
}

var packageManagers = []pkgManagerDef{
	{
		bin:    pmApt,
		phpPkg: func(branch string) string { return "php" + branch + "-cli" },
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmApt, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmApt, "remove", "-y", pkg}
		},
		preInstall: func(branch string) error {
			pkg := "php" + branch + "-cli"
			if isInstallable(pkg, pmApt, "-s", "install", pkg) {
				return nil
			}
			fmt.Println("Package not found, adding ondrej/php PPA...")
			add := exec.Command("sudo", "add-apt-repository", "-y", "ppa:ondrej/php")
			add.Stdout, add.Stderr = os.Stdout, os.Stderr
			if err := add.Run(); err != nil {
				return fmt.Errorf("add-apt-repository: %w", err)
			}
			upd := exec.Command("sudo", pmApt, "update")
			upd.Stdout, upd.Stderr = os.Stdout, os.Stderr
			return upd.Run()
		},
	},
	{
		bin:    pmDnf,
		phpPkg: func(branch string) string { return "php" + branch + "-php-cli" },
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmDnf, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmDnf, "remove", "-y", pkg}
		},
		preInstall: func(branch string) error {
			pkg := "php" + branch + "-php-cli"
			if isInstallable(pkg, pmDnf, "info", pkg) {
				return nil
			}
			fmt.Println("Package not found, adding Remi repository...")
			remi := exec.Command("sudo", pmDnf, "install", "-y",
				"https://rpms.remirepo.net/fedora/remi-release-$(rpm -E %fedora).rpm")
			remi.Stdout, remi.Stderr = os.Stdout, os.Stderr
			return remi.Run()
		},
	},
	{
		bin:    pmYum,
		phpPkg: func(branch string) string { return "php" + branch + "-php-cli" },
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmYum, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmYum, "remove", "-y", pkg}
		},
		preInstall: func(branch string) error {
			pkg := "php" + branch + "-php-cli"
			if isInstallable(pkg, pmYum, "info", pkg) {
				return nil
			}
			fmt.Println("Package not found, adding Remi repository...")
			remi := exec.Command("sudo", pmYum, "install", "-y",
				"https://rpms.remirepo.net/enterprise/remi-release-$(rpm -E %rhel).rpm")
			remi.Stdout, remi.Stderr = os.Stdout, os.Stderr
			return remi.Run()
		},
	},
	{
		bin:    pmPacman,
		phpPkg: func(branch string) string { return "php" },
		phpBin: func(branch string) string { return phpBinDir },
		installArgs: func(pkg string) []string {
			return []string{pmPacman, "-S", "--noconfirm", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmPacman, "-R", "--noconfirm", pkg}
		},
		preInstall: nil,
	},
	{
		bin:    pmZypper,
		phpPkg: func(branch string) string { return "php" + branch },
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmZypper, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmZypper, "remove", "-y", pkg}
		},
		preInstall: nil,
	},
}

func detectPackageManager() *pkgManagerDef {
	for i := range packageManagers {
		if _, err := exec.LookPath(packageManagers[i].bin); err == nil {
			return &packageManagers[i]
		}
	}
	return nil
}

func isInstallable(pkg string, args ...string) bool {
	return exec.Command(args[0], args[1:]...).Run() == nil
}

func LinuxInstall(base, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return fmt.Errorf("no supported package manager found (apt, dnf, yum, pacman, zypper)")
	}

	branch := majorMinor(ver)
	pkg := pm.phpPkg(branch)

	if pm.preInstall != nil {
		if err := pm.preInstall(branch); err != nil {
			return fmt.Errorf("prepare repository: %w", err)
		}
	}

	cmd := exec.Command("sudo", pm.installArgs(pkg)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install PHP %s via %s: %w", ver, pm.bin, err)
	}

	binPath := pm.phpBin(branch)
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}

	verDir := filepath.Join(base, "versions", ver)
	if err := os.MkdirAll(verDir, 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}

	return os.WriteFile(filepath.Join(verDir, "binary"), []byte(binPath), 0644)
}

func LinuxRemove(base, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return fmt.Errorf("no supported package manager found (apt, dnf, yum, pacman, zypper)")
	}

	branch := majorMinor(ver)
	pkg := pm.phpPkg(branch)

	cmd := exec.Command("sudo", pm.removeArgs(pkg)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remove PHP %s via %s: %w", ver, pm.bin, err)
	}
	return nil
}
