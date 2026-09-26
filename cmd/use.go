package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/project"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/system"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

var UseCmd = &cobra.Command{
	Use:     "use [u] [version|lts]",
	Aliases: []string{"u"},
	Short:   "Switch the global PHP version",
	Long: `Switch the global PHP version.

Without arguments, uses the version from the nearest .php-version file.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUse,
}

func runUse(cmd *cobra.Command, args []string) error {
	arg, err := useArg(args, ".", cmd.OutOrStdout())
	if err != nil {
		return err
	}

	return useVersion(
		arg,
		phpfs.NewManager(baseDir()),
		phpLTSResolver{ctx: cmd.Context()},
		cmd.OutOrStdout(),
	)
}

func useVersion(
	arg string,
	m *phpfs.Manager,
	r version.Resolver,
	out io.Writer,
) error {
	concrete, wasAlias, err := version.Resolve(arg, r)
	if err != nil {
		return err
	}

	if _, err := version.Parse(concrete); err != nil {
		return fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	if !m.VersionInstalled(concrete) {
		label := concrete
		if wasAlias {
			label = fmt.Sprintf("%s (lts)", concrete)
		}
		return fmt.Errorf("%s not installed — run: pvm install %s", label, arg)
	}

	binPath, err := m.GetVersionBinary(concrete)
	if err != nil {
		return err
	}

	if err := symlink.SetCurrent(m.Base, concrete, binPath); err != nil {
		return fmt.Errorf("activate PHP %s: %w", concrete, err)
	}

	label := concrete
	if wasAlias {
		label = fmt.Sprintf("%s (lts)", concrete)
	}
	fmt.Fprintf(out, "Now using PHP %s.\n", label)
	printPathHint(out, m.Base)
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

func printPathHint(out io.Writer, base string) {
	managed := symlink.ShimDir(base)

	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.EqualFold(p, managed) {
			return
		}
	}

	switch runtime.GOOS {
	case system.Windows:
		fmt.Fprintf(out, "\nOne-time setup: reload your PowerShell profile to activate version switching:\n")
		fmt.Fprintf(out, "  . $PROFILE\n")
		fmt.Fprintf(out, "\nAfter that, pvm use will switch versions instantly in any new terminal.\n")
	default:
		fmt.Fprintf(out, "\nHint: add %s to your PATH to use this version:\n", managed)
		fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", managed)
	}
}
