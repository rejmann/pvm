package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:     "list [ls]",
	Aliases: []string{"ls"},
	Short:   "List installed PHP versions",
	Args:    cobra.NoArgs,
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	return listVersions(newManager(cmd.Context()), cmd.OutOrStdout())
}

func listVersions(m *pvm.Manager, out io.Writer) error {
	l, err := m.List()
	if err != nil {
		return err
	}

	if len(l.Managed) > 0 {
		fmt.Fprintln(out, "pvm managed:")
		for _, v := range l.Managed {
			if v == l.Current {
				fmt.Fprintf(out, "  %s (current)\n", v)
			} else {
				fmt.Fprintf(out, "  %s\n", v)
			}
		}
	}

	if len(l.System) > 0 {
		fmt.Fprintln(out, "system:")
		for _, s := range l.System {
			fmt.Fprintf(out, "  %s  (%s)\n", s.Version, s.Binary)
		}
	}

	if len(l.Managed) == 0 && len(l.System) == 0 {
		fmt.Fprintln(out, "No PHP versions found.")
		fmt.Fprintln(out, "Run 'pvm available' to see installable versions.")
	}
	return nil
}
