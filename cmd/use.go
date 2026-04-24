package cmd

import (
	"fmt"
	"io"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

var UseCmd = &cobra.Command{
	Use:   "use <version|lts>",
	Short: "Switch to a PHP version",
	Args:  cobra.ExactArgs(1),
	RunE:  runUse,
}

func runUse(cmd *cobra.Command, args []string) error {
	return useVersion(
		args[0],
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
	return nil
}
