package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

func newCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "current [cur]",
		Aliases: []string{"cur"},
		Short:   "Show the currently active PHP version",
		Args:    cobra.NoArgs,
		RunE:    runCurrent,
	}
}

func runCurrent(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return printCurrent(newManager(cmd), dir, os.Getenv(pvm.EnvVersion), cmd.OutOrStdout())
}

func printCurrent(m *pvm.Manager, dir, env string, out io.Writer) error {
	a, err := m.Active(dir, env)
	if errors.Is(err, pvm.ErrNoActiveVersion) {
		fmt.Fprintln(out, "No PHP version is currently active.")
		return nil
	}
	if err != nil {
		return err
	}

	if a.Global() {
		fmt.Fprintf(out, "Current PHP version: %s\n", a.Version)
	} else {
		fmt.Fprintf(out, "Current PHP version: %s (set by %s)\n", a.Version, a.Source)
	}
	return nil
}
