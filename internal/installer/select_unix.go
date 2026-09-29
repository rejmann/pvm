//go:build linux || darwin

package installer

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/rejmann/pvm/internal/system"
)

func Install(base, ver string) error {
	switch runtime.GOOS {
	case system.Linux:
		return LinuxInstall(base, ver)
	case system.Darwin:
		return BrewInstall(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func Remove(base, ver string) error {
	switch runtime.GOOS {
	case system.Linux:
		return LinuxRemove(base, ver)
	case system.Darwin:
		return BrewRemove(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// EnsureExtensions installs the PHP extensions exts (e.g. "zip", "ext-xml")
// for an installed version.
func EnsureExtensions(base, ver string, exts []string) error {
	switch runtime.GOOS {
	case system.Linux:
		return LinuxEnsureExtensions(base, ver, exts)
	case system.Darwin:
		// Homebrew's php@X.Y bundles the common extensions; PECL ones are not managed yet.
		return fmt.Errorf("pvm cannot install PHP extensions with Homebrew yet (%s)", strings.Join(exts, ", "))
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
