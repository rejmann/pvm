package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rejmann/pvm/internal/home"
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
	return printWhich(home.Default(), dir, os.Getenv(envVersion), cmd.OutOrStdout())
}

func printWhich(h *home.Dir, dir, env string, out io.Writer) error {
	a, err := resolveActive(h, dir, env)
	if errors.Is(err, ErrNoActiveVersion) {
		return fmt.Errorf("%w — run: pvm use <version>", err)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(out, a.Binary)
	return nil
}
