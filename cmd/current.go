package cmd

import (
	"errors"
	"fmt"
	"io"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/symlink"
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
	return printCurrent(phpfs.NewManager(baseDir()), cmd.OutOrStdout())
}

func printCurrent(m *phpfs.Manager, out io.Writer) error {
	v, err := symlink.GetCurrent(m.Base)
	if err != nil {
		if errors.Is(err, symlink.ErrNoCurrentVersion) {
			fmt.Fprintln(out, "No PHP version is currently active.")
			return nil
		}
		return err
	}
	fmt.Fprintf(out, "Current PHP version: %s\n", v)
	return nil
}
