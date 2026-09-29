package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

type RemoverFunc func(h *home.Dir, ver string) error

var RemoveCmd = &cobra.Command{
	Use:     "remove [rm] <version>",
	Aliases: []string{"rm"},
	Short: "Remove an installed PHP version",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	return removeVersion(
		args[0],
		home.Default(),
		installer.Remove,
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
	)
}

func removeVersion(arg string, h *home.Dir, remove RemoverFunc, out, errOut io.Writer) error {
	if _, err := version.Parse(arg); err != nil {
		return fmt.Errorf("invalid version %q: %w", arg, err)
	}

	if !h.Installed(arg) {
		return fmt.Errorf("PHP %s is not installed", arg)
	}

	current, _ := h.Current()
	isCurrent := current == arg

	if err := remove(h, arg); err != nil {
		return fmt.Errorf("remove PHP %s: %w", arg, err)
	}

	if err := h.RemoveVersion(arg); err != nil {
		return fmt.Errorf("remove PHP %s metadata: %w", arg, err)
	}

	if err := composer.Remove(h.ComposerDir(), arg); err != nil {
		fmt.Fprintf(errOut, "Warning: could not remove Composer for PHP %s: %v\n", arg, err)
	}

	if isCurrent {
		_ = symlink.RemoveCurrent(h)
		fmt.Fprintf(errOut, "Warning: PHP %s was the active version. No version is now active.\n", arg)
	}

	fmt.Fprintf(out, "PHP %s removed.\n", arg)
	return nil
}
