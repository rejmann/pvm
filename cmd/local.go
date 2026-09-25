package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/project"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

var localUnset bool

var LocalCmd = &cobra.Command{
	Use:   "local [version|lts]",
	Short: "Set or show the PHP version for the current project (.php-version)",
	Long: `Set or show the PHP version for the current project.

With a version, writes it to .php-version in the current directory. From then
on, running php inside this directory (or any subdirectory) uses that version.
Without arguments, prints the version from the nearest .php-version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLocal,
}

func init() {
	LocalCmd.Flags().BoolVar(&localUnset, "unset", false, "Remove .php-version from the current directory")
}

func runLocal(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	m := phpfs.NewManager(baseDir())
	out := cmd.OutOrStdout()

	switch {
	case localUnset:
		if len(args) > 0 {
			return errors.New("--unset does not take a version")
		}
		return unsetLocal(dir, out)
	case len(args) == 0:
		return showLocal(dir, out)
	default:
		return setLocal(args[0], m, phpLTSResolver{ctx: cmd.Context()}, dir, out)
	}
}

func setLocal(arg string, m *phpfs.Manager, r version.Resolver, dir string, out io.Writer) error {
	concrete, _, err := version.Resolve(arg, r)
	if err != nil {
		return err
	}

	if _, err := version.Parse(concrete); err != nil {
		return fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	installed, ok := m.MatchInstalled(concrete)
	if !ok {
		return fmt.Errorf("PHP %s is not installed — run: pvm install %s", concrete, concrete)
	}

	path, err := project.Write(dir, concrete)
	if err != nil {
		return err
	}

	if err := symlink.EnsureShim(m.Base); err != nil {
		return fmt.Errorf("install php shim: %w", err)
	}

	fmt.Fprintf(out, "PHP %s will be used in %s (%s).\n", installed, dir, path)
	printPathHint(out, m.Base)
	return nil
}

func showLocal(dir string, out io.Writer) error {
	v, path, err := project.Find(dir)
	if errors.Is(err, project.ErrNotFound) {
		fmt.Fprintf(out, "No %s found in %s or any parent directory.\n", project.FileName, dir)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s (set by %s)\n", v, path)
	return nil
}

func unsetLocal(dir string, out io.Writer) error {
	path, err := project.Remove(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed %s.\n", path)
	return nil
}
