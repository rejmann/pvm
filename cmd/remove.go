package cmd

import (
	"fmt"
	"io"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

type RemoverFunc func(base, ver string) error

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
		phpfs.NewManager(baseDir()),
		installer.Remove,
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
	)
}

func removeVersion(arg string, m *phpfs.Manager, remove RemoverFunc, out, errOut io.Writer) error {
	if _, err := version.Parse(arg); err != nil {
		return fmt.Errorf("invalid version %q: %w", arg, err)
	}

	if !m.VersionInstalled(arg) {
		return fmt.Errorf("PHP %s is not installed", arg)
	}

	current, _ := symlink.GetCurrent(m.Base)
	isCurrent := current == arg

	if err := remove(m.Base, arg); err != nil {
		return fmt.Errorf("remove PHP %s: %w", arg, err)
	}

	if err := m.RemoveVersionDir(arg); err != nil {
		return fmt.Errorf("remove PHP %s metadata: %w", arg, err)
	}

	if isCurrent {
		_ = symlink.RemoveCurrent(m.Base)
		fmt.Fprintf(errOut, "Warning: PHP %s was the active version. No version is now active.\n", arg)
	}

	fmt.Fprintf(out, "PHP %s removed.\n", arg)
	return nil
}
