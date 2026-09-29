package symlink

import "github.com/rejmann/pvm/internal/home"

// Global switches the global PHP version: current-version plus the shim and,
// on Linux, update-alternatives.
type Global struct{}

func (Global) Activate(h *home.Dir, ver, bin string) error { return SetCurrent(h, ver, bin) }

func (Global) Deactivate(h *home.Dir) error { return RemoveCurrent(h) }
