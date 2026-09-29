package cmd

import (
	"context"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/sysphp"
)

// newManager wires the pvm use cases to this system.
func newManager(ctx context.Context) *pvm.Manager {
	return &pvm.Manager{
		Home:      home.Default(),
		Installer: installer.System{},
		Activator: symlink.Global{},
		LTS:       phpLTSResolver{ctx: ctx},
		System:    sysphp.Detect,
	}
}
