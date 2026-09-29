//go:build linux || darwin

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rejmann/pvm/internal/home"
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
	// extPkg names the package that provides a (normalized) PHP extension;
	// nil when pvm cannot install extensions with this package manager.
	extPkg func(branch, ext string) string
}

var packageManagers = []pkgManagerDef{
	{
		bin:       pmApt,
		phpPkg:    func(branch string) string { return "php" + branch + "-cli" },
		extPkg:    func(branch, ext string) string { return "php" + branch + "-" + ext },
		phpBin:    func(branch string) string { return phpBinDir + branch },
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
			// add-apt-repository already runs apt-get update; a second update
			// right after it can fail on the apt lists lock.
			if err := add.Run(); err != nil {
				return fmt.Errorf("add-apt-repository: %w", err)
			}
			return nil
		},
	},
	{
		bin:       pmDnf,
		phpPkg:    func(branch string) string { return "php" + branch + "-php-cli" },
		extPkg:    remiExtPkg,
		phpBin:    func(branch string) string { return phpBinDir + branch },
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
		bin:       pmYum,
		phpPkg:    func(branch string) string { return "php" + branch + "-php-cli" },
		extPkg:    remiExtPkg,
		phpBin:    func(branch string) string { return phpBinDir + branch },
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

func LinuxInstall(h *home.Dir, ver string) error {
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

	// pacman/zypper have no extPkg: pvm does not manage their extensions.
	if pm.extPkg != nil {
		if err := installExtensions(pm, h, ver, BaseExtensions); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v. PHP works; pvm composer offers to install missing extensions when a project needs them.\n", err)
		}
	}

	binPath := pm.phpBin(branch)
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}

	return h.SetBinary(ver, binPath)
}

func LinuxRemove(h *home.Dir, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return fmt.Errorf("no supported package manager found (apt, dnf, yum, pacman, zypper)")
	}

	branch := majorMinor(ver)
	pkg := pm.phpPkg(branch)

	// Extensions first, and quietly: some may never have been installed. The
	// base ones are always included, for versions installed before pvm
	// recorded its packages.
	extras := h.Packages(ver)
	if pm.extPkg != nil {
		for _, ext := range BaseExtensions {
			extras = append(extras, pm.extPkg(branch, ext))
		}
	}
	for _, extra := range extras {
		exec.Command("sudo", pm.removeArgs(extra)...).Run()
	}

	cmd := exec.Command("sudo", pm.removeArgs(pkg)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remove PHP %s via %s: %w", ver, pm.bin, err)
	}
	return nil
}

// remiExtPkg names Remi's package for ext; PECL extensions carry a pecl- prefix.
func remiExtPkg(branch, ext string) string {
	switch ext {
	case "mysql":
		ext = "mysqlnd"
	case "sqlite3":
		ext = "pdo"
	case "zip", "redis", "xdebug", "imagick", "apcu", "memcached", "igbinary", "mongodb":
		ext = "pecl-" + ext
	}
	return "php" + branch + "-php-" + ext
}

// installExtensions installs the packages of exts one by one and records the
// ones that succeed, so LinuxRemove removes them with the version.
func installExtensions(pm *pkgManagerDef, h *home.Dir, ver string, exts []string) error {
	if pm.extPkg == nil {
		return fmt.Errorf("pvm cannot install PHP extensions with %s", pm.bin)
	}
	branch := majorMinor(ver)

	var installed, failed []string
	seen := map[string]bool{}
	for _, ext := range exts {
		pkg := pm.extPkg(branch, normalizeExtension(ext))
		if seen[pkg] {
			continue
		}
		seen[pkg] = true

		cmd := exec.Command("sudo", pm.installArgs(pkg)...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			failed = append(failed, pkg)
			continue
		}
		installed = append(installed, pkg)
	}

	if err := h.AddPackages(ver, installed); err != nil {
		return fmt.Errorf("record installed extensions: %w", err)
	}
	if len(failed) > 0 {
		return fmt.Errorf("could not install %s via %s", strings.Join(failed, ", "), pm.bin)
	}
	return nil
}

// LinuxEnsureExtensions installs the packages of exts for an installed version.
func LinuxEnsureExtensions(h *home.Dir, ver string, exts []string) error {
	pm := detectPackageManager()
	if pm == nil {
		return fmt.Errorf("no supported package manager found (apt, dnf, yum, pacman, zypper)")
	}
	return installExtensions(pm, h, ver, exts)
}
