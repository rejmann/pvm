package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/cmd/use"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use [u] [version|lts]",
		Aliases: []string{"u"},
		Short:   "Switch the global PHP version",
		Long: `Switch the global PHP version.

Without arguments, uses the version from the nearest .php-version file.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUse,
	}
}

func runUse(cmd *cobra.Command, args []string) error {
	arg, err := use.Arg(args, ".", cmd.OutOrStdout())
	if err != nil {
		return err
	}
	return useVersion(newManager(cmd), arg, cmd.OutOrStdout())
}

func useVersion(m *pvm.Manager, arg string, out io.Writer) error {
	t, err := m.Target(arg)
	if err != nil {
		return err
	}
	if err := m.Use(t); err != nil {
		return err
	}

	fmt.Fprintf(out, "Now using PHP %s.\n", t)
	use.PrintPathHint(out, m.Home)
	return nil
}
