package cmd

import (
	"os"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/shim"
	"github.com/rejmann/pvm/internal/sysphp"
	"github.com/spf13/cobra"
)

// newManager wires the pvm use cases to this system, with the output of the
// tools pvm runs going to the command's stdout and stderr.
func newManager(cmd *cobra.Command) *pvm.Manager {
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	h := home.Default()
	return &pvm.Manager{
		Home:      h,
		Installer: installer.New(out, errOut),
		Activator: shim.New(out, errOut),
		LTS:       phpLTSResolver{ctx: cmd.Context()},
		System:    sysphp.Detect,
		SystemPHP: func() (sysphp.PHP, bool) { return sysphp.OnPath(os.Getenv("PATH"), h.ShimDir()) },
	}
}
