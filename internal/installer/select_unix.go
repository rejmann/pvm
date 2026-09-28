//go:build linux || darwin

package installer

import (
	"fmt"
	"runtime"

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

// EnsureExtensions adds the PHP extensions pvm installs alongside PHP (zip,
// for Composer) to an already installed version.
func EnsureExtensions(base, ver string) error {
	switch runtime.GOOS {
	case system.Linux:
		return LinuxEnsureExtensions(base, ver)
	case system.Darwin:
		return nil // Homebrew's php@X.Y already includes zip
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
