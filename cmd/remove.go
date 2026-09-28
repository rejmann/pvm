package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/activate"
	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

// removeFunc uninstalls PHP ver; tests replace installer.Remove with it.
type removeFunc func(ver string) error

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove [rm] <version>",
		Aliases: []string{"rm"},
		Short:   "Remove an installed PHP version",
		Args:    cobra.ExactArgs(1),
		RunE:    runRemove,
	}
}

func runRemove(cmd *cobra.Command, args []string) error {
	h := home.Default()
	remove := func(ver string) error {
		return installer.Remove(h, ver, streams(cmd))
	}
	return removeVersion(args[0], h, remove, cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func removeVersion(arg string, h *home.Dir, remove removeFunc, out, errOut io.Writer) error {
	if _, err := version.Parse(arg); err != nil {
		return fmt.Errorf("invalid version %q: %w", arg, err)
	}

	if !h.VersionInstalled(arg) {
		return fmt.Errorf("PHP %s is not installed", arg)
	}

	current, _ := h.Current()
	isCurrent := current == arg

	if err := remove(arg); err != nil {
		return fmt.Errorf("remove PHP %s: %w", arg, err)
	}

	if err := h.RemoveVersionDir(arg); err != nil {
		return fmt.Errorf("remove PHP %s metadata: %w", arg, err)
	}

	if err := composer.Remove(h.ComposerDir(), arg); err != nil {
		fmt.Fprintf(errOut, "Warning: could not remove Composer for PHP %s: %v\n", arg, err)
	}

	if isCurrent {
		_ = activate.Clear(h, proc.Streams{Out: out, Err: errOut})
		fmt.Fprintf(errOut, "Warning: PHP %s was the active version. No version is now active.\n", arg)
	}

	fmt.Fprintf(out, "PHP %s removed.\n", arg)
	return nil
}
