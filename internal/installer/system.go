package installer

import "github.com/rejmann/pvm/internal/home"

// System installs PHP the way this OS supports: a package manager on Linux,
// Homebrew on macOS, windows.php.net builds on Windows.
type System struct{}

func (System) Install(h *home.Dir, ver string) error { return Install(h, ver) }

func (System) Remove(h *home.Dir, ver string) error { return Remove(h, ver) }
