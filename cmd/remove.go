package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

var RemoveCmd = &cobra.Command{
	Use:     "remove [rm] <version>",
	Aliases: []string{"rm"},
	Short:   "Remove an installed PHP version",
	Args:    cobra.ExactArgs(1),
	RunE:    runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	return removeVersion(newManager(cmd.Context()), args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func removeVersion(m *pvm.Manager, v string, out, errOut io.Writer) error {
	r, err := m.Remove(v)
	if err != nil {
		return err
	}
	if r.ComposerErr != nil {
		fmt.Fprintf(errOut, "Warning: could not remove Composer for PHP %s: %v\n", v, r.ComposerErr)
	}
	if r.WasCurrent {
		fmt.Fprintf(errOut, "Warning: PHP %s was the active version. No version is now active.\n", v)
	}
	fmt.Fprintf(out, "PHP %s removed.\n", v)
	return nil
}
