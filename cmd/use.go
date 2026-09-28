package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rejmann/pvm/internal/activate"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/project"
	"github.com/rejmann/pvm/internal/version"
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

	return useVersion(arg, home.Default(), ltsResolver{ctx: cmd.Context()}, streams(cmd))
}

func useVersion(arg string, h *home.Dir, r version.Resolver, s proc.Streams) error {
	concrete, wasAlias, err := version.Resolve(arg, r)
	if err != nil {
		return err
	}

	if _, err := version.Parse(concrete); err != nil {
		return fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	label := versionLabel(concrete, wasAlias)
	if !h.VersionInstalled(concrete) {
		return fmt.Errorf("%s not installed — run: pvm install %s", label, arg)
	}

	bin, err := h.VersionBinary(concrete)
	if err != nil {
		return err
	}

	if err := activate.Use(h, concrete, bin, s); err != nil {
		return fmt.Errorf("activate PHP %s: %w", concrete, err)
	}

	fmt.Fprintf(s.Out, "Now using PHP %s.\n", label)
	printPathHint(s.Out, h)
	return nil
}

// useArg returns the version to switch to: the explicit argument, or the one
// declared in the nearest .php-version when none is given.
func useArg(args []string, dir string, out io.Writer) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	v, path, err := project.Find(dir)
	if errors.Is(err, project.ErrNotFound) {
		return "", fmt.Errorf("no version given and no %s found — run: pvm use <version>", project.FileName)
	}
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

	if runtime.GOOS == "windows" {
		fmt.Fprintf(out, "\nOne-time setup: reload your PowerShell profile to activate version switching:\n")
		fmt.Fprintf(out, "  . $PROFILE\n")
		fmt.Fprintf(out, "\nAfter that, pvm use will switch versions instantly in any new terminal.\n")
		return
	}
	fmt.Fprintf(out, "\nHint: add %s to your PATH to use this version:\n", managed)
	fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", managed)
}
