//go:build linux

package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
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
	preInstall  func(branch string, s proc.Streams) error
	// extraPkgs are extensions Composer needs that the -cli package leaves out
	// (zip, to extract packages without unzip on the system). Best effort: a
	// failure only prints a warning.
	extraPkgs func(branch string) []string
}

var packageManagers = []pkgManagerDef{
	{
		bin:       pmApt,
		phpPkg:    func(branch string) string { return "php" + branch + "-cli" },
		extraPkgs: func(branch string) []string { return []string{"php" + branch + "-zip"} },
		phpBin:    func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmApt, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmApt, "remove", "-y", pkg}
		},
		preInstall: func(branch string, s proc.Streams) error {
			pkg := "php" + branch + "-cli"
			if proc.Quiet(pmApt, "-s", "install", pkg) == nil {
				return nil
			}
			fmt.Fprintln(s.Out, "Package not found, adding ondrej/php PPA...")
			// add-apt-repository already runs apt-get update; a second update
			// right after it can fail on the apt lists lock.
			if err := s.Run("sudo", "add-apt-repository", "-y", "ppa:ondrej/php"); err != nil {
				return fmt.Errorf("add-apt-repository: %w", err)
			}
			return nil
		},
	},
	{
		bin:       pmDnf,
		phpPkg:    func(branch string) string { return "php" + branch + "-php-cli" },
		extraPkgs: func(branch string) []string { return []string{"php" + branch + "-php-pecl-zip"} },
		phpBin:    func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmDnf, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmDnf, "remove", "-y", pkg}
		},
		preInstall: func(branch string, s proc.Streams) error {
			pkg := "php" + branch + "-php-cli"
			if proc.Quiet(pmDnf, "info", pkg) == nil {
				return nil
			}
			fmt.Fprintln(s.Out, "Package not found, adding Remi repository...")
			return s.Run("sudo", pmDnf, "install", "-y",
				"https://rpms.remirepo.net/fedora/remi-release-$(rpm -E %fedora).rpm")
		},
	},
	{
		bin:       pmYum,
		phpPkg:    func(branch string) string { return "php" + branch + "-php-cli" },
		extraPkgs: func(branch string) []string { return []string{"php" + branch + "-php-pecl-zip"} },
		phpBin:    func(branch string) string { return phpBinDir + branch },
		installArgs: func(pkg string) []string {
			return []string{pmYum, "install", "-y", pkg}
		},
		removeArgs: func(pkg string) []string {
			return []string{pmYum, "remove", "-y", pkg}
		},
		preInstall: func(branch string, s proc.Streams) error {
			pkg := "php" + branch + "-php-cli"
			if proc.Quiet(pmYum, "info", pkg) == nil {
				return nil
			}
			fmt.Fprintln(s.Out, "Package not found, adding Remi repository...")
			return s.Run("sudo", pmYum, "install", "-y",
				"https://rpms.remirepo.net/enterprise/remi-release-$(rpm -E %rhel).rpm")
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

var errNoPackageManager = errors.New("no supported package manager found (apt, dnf, yum, pacman, zypper)")

func detectPackageManager() *pkgManagerDef {
	for i := range packageManagers {
		if _, err := exec.LookPath(packageManagers[i].bin); err == nil {
			return &packageManagers[i]
		}
	}
	return nil
}

// Install installs PHP ver with the system package manager and returns its
// php binary.
func Install(ctx context.Context, h *home.Dir, ver string, s proc.Streams) (string, error) {
	pm := detectPackageManager()
	if pm == nil {
		return "", errNoPackageManager
	}

	branch := version.Branch(ver)
	pkg := pm.phpPkg(branch)

	if pm.preInstall != nil {
		if err := pm.preInstall(branch, s); err != nil {
			return "", fmt.Errorf("prepare repository: %w", err)
		}
	}

	if err := s.Run("sudo", pm.installArgs(pkg)...); err != nil {
		return "", fmt.Errorf("install PHP %s via %s: %w", ver, pm.bin, err)
	}

	if err := installExtras(pm, branch, s); err != nil {
		fmt.Fprintf(s.Err, "Warning: %v. PHP works, but Composer will need unzip or 7z to extract packages; pvm composer offers to retry.\n", err)
	}

	bin := pm.phpBin(branch)
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("PHP binary not found at %s after installation", bin)
	}
	return bin, nil
}

func Remove(h *home.Dir, ver string, s proc.Streams) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}

	branch := version.Branch(ver)
	pkg := pm.phpPkg(branch)

	// Extras first, and quietly: they may never have been installed.
	for _, extra := range extras(pm, branch) {
		proc.Quiet("sudo", pm.removeArgs(extra)...)
	}

	if err := s.Run("sudo", pm.removeArgs(pkg)...); err != nil {
		return fmt.Errorf("remove PHP %s via %s: %w", ver, pm.bin, err)
	}
	return nil
}

func extras(pm *pkgManagerDef, branch string) []string {
	if pm.extraPkgs == nil {
		return nil
	}
	return pm.extraPkgs(branch)
}

// installExtras installs pm's extra packages one by one. Install only warns
// on failure: PHP itself is already installed and works without them.
func installExtras(pm *pkgManagerDef, branch string, s proc.Streams) error {
	var failed []string
	for _, pkg := range extras(pm, branch) {
		if err := s.Run("sudo", pm.installArgs(pkg)...); err != nil {
			failed = append(failed, pkg)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("could not install %s via %s", strings.Join(failed, ", "), pm.bin)
	}
	return nil
}

// EnsureExtensions adds the extensions pvm installs alongside PHP (zip, for
// Composer) to an already installed version.
func EnsureExtensions(h *home.Dir, ver string, s proc.Streams) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}
	if pm.extraPkgs == nil {
		return fmt.Errorf("pvm cannot add PHP extensions with %s", pm.bin)
	}
	return installExtras(pm, version.Branch(ver), s)
}
