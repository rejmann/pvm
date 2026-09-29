//go:build linux

package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/sudo"
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
			add := sudo.Command("add-apt-repository", "-y", "ppa:ondrej/php")
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
			return addRemi(pmDnf, "fedora", "%fedora", stdout, stderr)
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
			return addRemi(pmYum, "enterprise", "%rhel", stdout, stderr)
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
		// openSUSE names extension packages by major version only (php8-intl).
		extPkg: func(branch, ext string) string {
			major, _, _ := strings.Cut(branch, ".")
			return "php" + major + "-" + ext
		},
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

// addRemi installs the Remi release package with pm; macro (%fedora or
// %rhel) is expanded by rpm, since no shell runs the command.
func addRemi(pm, dist, macro string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stdout, "Package not found, adding Remi repository...")
	out, err := exec.Command("rpm", "-E", macro).Output()
	if err != nil {
		return fmt.Errorf("rpm -E %s: %w", macro, err)
	}
	release := strings.TrimSpace(string(out))
	remi := sudo.Command(pm, "install", "-y",
		"https://rpms.remirepo.net/"+dist+"/remi-release-"+release+".rpm")
	remi.Stdout, remi.Stderr = stdout, stderr
	return remi.Run()
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
	cmd := sudo.Command(args...)
	cmd.Stdout, cmd.Stderr = s.Stdout, s.Stderr
	return cmd.Run()
}

// Install installs PHP ver and the base extensions, and records its binary.
func (s *System) Install(h *home.Dir, ver string) error {
	pm := detectPackageManager()
	if pm == nil {
		return errNoPackageManager
	}

	if err := sudo.Authenticate(s.Stderr, "install PHP "+ver+" with "+pm.bin); err != nil {
		return err
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
	if err := sudo.Authenticate(s.Stderr, "remove PHP "+ver+" with "+pm.bin); err != nil {
		return err
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
		sudo.Command(pm.removeArgs(extra)...).Run()
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
	if pm.extPkg == nil {
		return fmt.Errorf("pvm cannot install PHP extensions with %s", pm.bin)
	}
	if err := sudo.Authenticate(s.Stderr, "install PHP extensions with "+pm.bin); err != nil {
		return err
	}
	return s.installExtensions(pm, h, ver, exts)
}

// RemoveExtensions uninstalls the packages pvm installed for exts (the
// recorded ones and the base extensions pvm install adds). The others ship
// with PHP, like calendar in php<X.Y>-common, or came from elsewhere: they
// are disabled instead, since uninstalling their package would take PHP (or
// something pvm does not own) with it.
func (s *System) RemoveExtensions(h *home.Dir, ver string, exts []string) (phpext.Removal, error) {
	var r phpext.Removal
	pm := detectPackageManager()
	if pm == nil {
		return r, errNoPackageManager
	}
	branch := version.Branch(ver)

	owned := map[string]bool{}
	for _, pkg := range h.Packages(ver) {
		owned[pkg] = true
	}
	if pm.extPkg != nil {
		for _, ext := range phpext.Base {
			owned[pm.extPkg(branch, ext)] = true
		}
	}

	var pkgs, others []string
	for _, ext := range exts {
		if pm.extPkg == nil {
			others = append(others, ext)
			continue
		}
		pkg := pm.extPkg(branch, phpext.Normalize(ext))
		if !owned[pkg] {
			others = append(others, ext)
			continue
		}
		r.Uninstalled = append(r.Uninstalled, ext)
		if !slices.Contains(pkgs, pkg) {
			pkgs = append(pkgs, pkg)
		}
	}

	var toDisable []string
	for _, ext := range others {
		if s.toggleable(pm, h, ver, ext) {
			toDisable = append(toDisable, ext)
		} else {
			r.Stuck = append(r.Stuck, ext)
		}
	}

	if len(pkgs) > 0 {
		if err := sudo.Authenticate(s.Stderr, "remove PHP extensions with "+pm.bin); err != nil {
			return phpext.Removal{}, err
		}
		var removed, failed []string
		for _, pkg := range pkgs {
			if err := s.sudo(pm.removeArgs(pkg)); err != nil {
				failed = append(failed, pkg)
				continue
			}
			removed = append(removed, pkg)
		}
		if err := h.RemovePackages(ver, removed); err != nil {
			return phpext.Removal{}, fmt.Errorf("record removed extensions: %w", err)
		}
		if len(failed) > 0 {
			return phpext.Removal{}, fmt.Errorf("could not remove %s via %s", strings.Join(failed, ", "), pm.bin)
		}
	}
	if len(toDisable) > 0 {
		if err := s.SetExtensionsEnabled(h, ver, toDisable, false); err != nil {
			return phpext.Removal{Uninstalled: r.Uninstalled}, err
		}
		r.Disabled = toDisable
	}
	return r, nil
}

// toggleable reports whether ext can be turned on and off for version ver:
// apt has its mods-available .ini, elsewhere a file of PHP's scan directory
// loads it. Extensions compiled into PHP have neither.
func (s *System) toggleable(pm *pkgManagerDef, h *home.Dir, ver, ext string) bool {
	if pm.bin == pmApt {
		_, err := os.Stat(aptModIni(version.Branch(ver), ext))
		return err == nil
	}
	bin, err := h.Binary(ver)
	if err != nil {
		return false
	}
	return confdLoads(bin, ext)
}

// aptModIni is the .ini of ext that phpenmod/phpdismod link into conf.d.
func aptModIni(branch, ext string) string {
	return "/etc/php/" + branch + "/mods-available/" + phpext.Name(ext) + ".ini"
}

// SetExtensionsEnabled turns exts on or off for installed version ver
// without uninstalling them: with phpenmod/phpdismod on Debian and Ubuntu,
// elsewhere by renaming the .ini file that loads each one.
func (s *System) SetExtensionsEnabled(h *home.Dir, ver string, exts []string, enabled bool) error {
	pm := detectPackageManager()
	if pm == nil || pm.bin != pmApt {
		bin, err := h.Binary(ver)
		if err != nil {
			return err
		}
		return setConfdEnabled(bin, exts, enabled, s.Stderr)
	}

	branch := version.Branch(ver)
	var missing []string
	for _, ext := range exts {
		if _, err := os.Stat(aptModIni(branch, ext)); err != nil {
			missing = append(missing, ext)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s not installed for PHP %s", strings.Join(missing, ", "), ver)
	}

	tool := "phpdismod"
	if enabled {
		tool = "phpenmod"
	}
	if err := sudo.Authenticate(s.Stderr, "run "+tool); err != nil {
		return err
	}
	return s.sudo(append([]string{tool, "-v", branch}, exts...))
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
