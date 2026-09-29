//go:build linux

package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/version"
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
	preInstall  func(branch string, stdout, stderr io.Writer) error
	// extPkg names the package that provides a (normalized) PHP extension;
	// nil when pvm cannot install extensions with this package manager.
	extPkg func(branch, ext string) string
}

var packageManagers = []pkgManagerDef{
	{
		bin:    pmApt,
		phpPkg: func(branch string) string { return "php" + branch + "-cli" },
		extPkg: func(branch, ext string) string { return "php" + branch + "-" + ext },
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmApt, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmApt, "remove", "-y", pkg}
		},
		preInstall: func(branch string, stdout, stderr io.Writer) error {
			pkg := "php" + branch + "-cli"
			if isInstallable(pkg, pmApt, "-s", "install", pkg) {
				return nil
			}
			fmt.Fprintln(stdout, "Package not found, adding ondrej/php PPA...")
			add := exec.Command("sudo", "add-apt-repository", "-y", "ppa:ondrej/php")
			add.Stdout, add.Stderr = stdout, stderr
			// add-apt-repository already runs apt-get update; a second update
			// right after it can fail on the apt lists lock.
			if err := add.Run(); err != nil {
				return fmt.Errorf("add-apt-repository: %w", err)
			}
			return nil
		},
	},
	{
		bin:    pmDnf,
		phpPkg: func(branch string) string { return "php" + branch + "-php-cli" },
		extPkg: remiExtPkg,
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmDnf, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmDnf, "remove", "-y", pkg}
		},
		preInstall: func(branch string, stdout, stderr io.Writer) error {
			pkg := "php" + branch + "-php-cli"
			if isInstallable(pkg, pmDnf, "info", pkg) {
				return nil
			}
			fmt.Fprintln(stdout, "Package not found, adding Remi repository...")
			remi := exec.Command("sudo", pmDnf, "install", "-y",
				"https://rpms.remirepo.net/fedora/remi-release-$(rpm -E %fedora).rpm")
			remi.Stdout, remi.Stderr = stdout, stderr
			return remi.Run()
		},
	},
	{
		bin:    pmYum,
		phpPkg: func(branch string) string { return "php" + branch + "-php-cli" },
		extPkg: remiExtPkg,
		phpBin: func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmYum, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmYum, "remove", "-y", pkg}
		},
		preInstall: func(branch string, stdout, stderr io.Writer) error {
			pkg := "php" + branch + "-php-cli"
			if isInstallable(pkg, pmYum, "info", pkg) {
				return nil
			}
			fmt.Fprintln(stdout, "Package not found, adding Remi repository...")
			remi := exec.Command("sudo", pmYum, "install", "-y",
				"https://rpms.remirepo.net/enterprise/remi-release-$(rpm -E %rhel).rpm")
			remi.Stdout, remi.Stderr = stdout, stderr
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

// System installs PHP with the distribution's package manager (apt, dnf,
// yum, pacman or zypper), through sudo.
type System struct {
	Stdout, Stderr io.Writer // where the package manager's output goes
}

// New returns the installer for this system.
func New(stdout, stderr io.Writer) *System {
	return &System{Stdout: stdout, Stderr: stderr}
}

var errNoPackageManager = fmt.Errorf("no supported package manager found (apt, dnf, yum, pacman, zypper)")

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

// sudo runs args as root with the package manager's output shown.
func (s *System) sudo(args []string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
	return cmd.Run()
}

// Install installs PHP ver and the base extensions, and records its binary.
func (s *System) Install(h *home.Dir, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}

	branch := version.Branch(ver)
	if pm.preInstall != nil {
		if err := pm.preInstall(branch, s.Stdout, s.Stderr); err != nil {
			return fmt.Errorf("prepare repository: %w", err)
		}
	}
	if err := s.sudo(pm.installArgs(pm.phpPkg(branch))); err != nil {
		return fmt.Errorf("install PHP %s via %s: %w", ver, pm.bin, err)
	}

	// pacman/zypper have no extPkg: pvm does not manage their extensions.
	if pm.extPkg != nil {
		if err := s.installExtensions(pm, h, ver, phpext.Base); err != nil {
			fmt.Fprintf(s.Stderr, "Warning: %v. PHP works; pvm composer offers to install missing extensions when a project needs them.\n", err)
		}
	}

	binPath := pm.phpBin(branch)
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("PHP binary not found at %s after installation", binPath)
	}
	return h.SetBinary(ver, binPath)
}

// Remove uninstalls PHP ver with the extension packages pvm installed for it.
func (s *System) Remove(h *home.Dir, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}
	branch := version.Branch(ver)

	// Extensions first, and quietly: some may never have been installed. The
	// base ones are always included, for versions installed before pvm
	// recorded its packages.
	extras := h.Packages(ver)
	if pm.extPkg != nil {
		for _, ext := range phpext.Base {
			extras = append(extras, pm.extPkg(branch, ext))
		}
	}
	for _, extra := range extras {
		exec.Command("sudo", pm.removeArgs(extra)...).Run()
	}

	if err := s.sudo(pm.removeArgs(pm.phpPkg(branch))); err != nil {
		return fmt.Errorf("remove PHP %s via %s: %w", ver, pm.bin, err)
	}
	return nil
}

// AddExtensions installs the packages of exts for installed version ver.
func (s *System) AddExtensions(h *home.Dir, ver string, exts []string) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}
	return s.installExtensions(pm, h, ver, exts)
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
// ones that succeed, so Remove removes them with the version.
func (s *System) installExtensions(pm *pkgManagerDef, h *home.Dir, ver string, exts []string) error {
	if pm.extPkg == nil {
		return fmt.Errorf("pvm cannot install PHP extensions with %s", pm.bin)
	}
	branch := version.Branch(ver)

	var installed, failed []string
	seen := map[string]bool{}
	for _, ext := range exts {
		pkg := pm.extPkg(branch, phpext.Normalize(ext))
		if seen[pkg] {
			continue
		}
		seen[pkg] = true

		if err := s.sudo(pm.installArgs(pkg)); err != nil {
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
