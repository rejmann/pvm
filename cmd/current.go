package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rejmann/pvm/internal/home"
	"github.com/spf13/cobra"
)

var CurrentCmd = &cobra.Command{
	Use:     "current [cur]",
	Aliases: []string{"cur"},
	Short:   "Show the currently active PHP version",
	Args:    cobra.NoArgs,
	RunE:    runCurrent,
}

func runCurrent(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return printCurrent(home.Default(), dir, os.Getenv(envVersion), cmd.OutOrStdout())
}

func printCurrent(h *home.Dir, dir, env string, out io.Writer) error {
	a, err := resolveActive(h, dir, env)
	if err != nil {
		if errors.Is(err, ErrNoActiveVersion) {
			fmt.Fprintln(out, "No PHP version is currently active.")
			return nil
		}
		return err
	}

	if a.Source == "global" {
		fmt.Fprintf(out, "Current PHP version: %s\n", a.Version)
	} else {
		fmt.Fprintf(out, "Current PHP version: %s (set by %s)\n", a.Version, a.Source)
	}
	return nil
}
