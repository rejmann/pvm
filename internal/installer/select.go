package installer

import (
	"fmt"
	"runtime"

	"github.com/rejmann/pvm/system"
)

func Install(base, ver string) error {
	switch runtime.GOOS {
	case system.Linux:
		return AptInstall(base, ver)
	case system.Darwin:
		return BrewInstall(base, ver)
	case system.Windows:
		return WindowsInstall(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func Remove(base, ver string) error {
	switch runtime.GOOS {
	case system.Linux:
		return AptRemove(base, ver)
	case system.Darwin:
		return BrewRemove(base, ver)
	case system.Windows:
		return WindowsRemove(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
