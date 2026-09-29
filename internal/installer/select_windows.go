//go:build windows

package installer

import (
	"fmt"
	"runtime"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/system"
)

func Install(h *home.Dir, ver string) error {
	switch runtime.GOOS {
	case system.Windows:
		return WindowsInstall(h, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func Remove(h *home.Dir, ver string) error {
	switch runtime.GOOS {
	case system.Windows:
		return WindowsRemove(h, ver)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// EnsureExtensions enables the PHP extensions exts (e.g. "zip", "ext-xml")
// for an installed version, in the php.ini pvm manages.
func EnsureExtensions(h *home.Dir, ver string, exts []string) error {
	switch runtime.GOOS {
	case system.Windows:
		return WindowsEnsureExtensions(h, ver, exts)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
