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
