package cmd

import (
	"fmt"
	"io"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

type InstallerFunc func(base, ver string) error

var InstallCmd = &cobra.Command{
	Use:     "install [i] <version|lts>",
	Aliases: []string{"i"},
	Short: "Install a PHP version",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	return installVersion(
		args[0],
		phpfs.NewManager(baseDir()),
		phpLTSResolver{ctx: cmd.Context()},
		installer.Install,
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
	)
}

func installVersion(
	arg string,
	m *phpfs.Manager,
	r version.Resolver,
	install InstallerFunc,
	out,
	errOut io.Writer,
) error {
	concrete, wasAlias, err := version.Resolve(arg, r)
	if err != nil {
		return err
	}

	if _, err := version.Parse(concrete); err != nil {
		return fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	if err := m.EnsurebaseDir(); err != nil {
		return fmt.Errorf("initialize \"pvm\" directory: %w", err)
	}

	if m.VersionInstalled(concrete) {
		label := concrete
		if wasAlias {
			label = fmt.Sprintf("%s (lts)", concrete)
		}
		return fmt.Errorf("%s already installed", label)
	}

	label := concrete
	if wasAlias {
		label = fmt.Sprintf("%s (lts)", concrete)
	}
	fmt.Fprintf(out, "Installing PHP %s...\n", label)

	if err := install(m.Base, concrete); err != nil {
		return fmt.Errorf("install PHP %s: %w", concrete, err)
	}

	fmt.Fprintf(out, "PHP %s installed successfully.\n", label)

	return nil
}
