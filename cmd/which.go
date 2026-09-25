package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/spf13/cobra"
)

var WhichCmd = &cobra.Command{
	Use:   "which",
	Short: "Print the path of the PHP binary used in the current directory",
	Args:  cobra.NoArgs,
	RunE:  runWhich,
}

func runWhich(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return printWhich(phpfs.NewManager(baseDir()), dir, os.Getenv(envVersion), cmd.OutOrStdout())
}

func printWhich(m *phpfs.Manager, dir, env string, out io.Writer) error {
	a, err := resolveActive(m, dir, env)
	if errors.Is(err, ErrNoActiveVersion) {
		return fmt.Errorf("%w — run: pvm use <version> or pvm local <version>", err)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(out, a.Binary)
	return nil
}
