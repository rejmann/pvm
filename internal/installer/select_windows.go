//go:build windows

package installer

import (
	"fmt"
	"runtime"

	"github.com/rejmann/pvm/internal/system"
)

func Install(base, ver string) error {
	switch runtime.GOOS {
	case system.Windows:
		return WindowsInstall(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func Remove(base, ver string) error {
	switch runtime.GOOS {
	case system.Windows:
		return WindowsRemove(base, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
