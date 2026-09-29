package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
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
	arg, err := useArg(args, ".", cmd.OutOrStdout())
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
	printPathHint(out, m.Home)
	return nil
}

// useArg returns the version to switch to: the explicit argument, or the one
// declared in the nearest .php-version when none is given.
func useArg(args []string, dir string, out io.Writer) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	v, path, err := pvm.ProjectVersion(dir)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(out, "Found %s with version %s.\n", path, v)
	return v, nil
}

func printPathHint(out io.Writer, h *home.Dir) {
	managed := h.ShimDir()

	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.EqualFold(p, managed) {
			return
		}
	}

	printPathSetup(out, managed)
}
