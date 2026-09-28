package cmd

import (
	"fmt"
	"io"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

// installFunc installs PHP ver and returns its php binary; tests replace
// installer.Install with it.
type installFunc func(ver string) (bin string, err error)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "install [i] <version|lts>",
		Aliases: []string{"i"},
		Short:   "Install a PHP version",
		Args:    cobra.ExactArgs(1),
		RunE:    runInstall,
	}
}

func runInstall(cmd *cobra.Command, args []string) error {
	h := home.Default()
	install := func(ver string) (string, error) {
		return installer.Install(cmd.Context(), h, ver, streams(cmd))
	}
	return installVersion(args[0], h, ltsResolver{ctx: cmd.Context()}, install, cmd.OutOrStdout())
}

func installVersion(arg string, h *home.Dir, r version.Resolver, install installFunc, out io.Writer) error {
	concrete, wasAlias, err := version.Resolve(arg, r)
	if err != nil {
		return err
	}

	if _, err := version.Parse(concrete); err != nil {
		return fmt.Errorf("invalid version %q: %w", concrete, err)
	}

	if err := h.EnsureBaseDir(); err != nil {
		return fmt.Errorf("initialize \"pvm\" directory: %w", err)
	}

	label := versionLabel(concrete, wasAlias)
	if h.VersionInstalled(concrete) {
		return fmt.Errorf("%s already installed", label)
	}

	fmt.Fprintf(out, "Installing PHP %s...\n", label)

	bin, err := install(concrete)
	if err != nil {
		return fmt.Errorf("install PHP %s: %w", concrete, err)
	}
	if err := h.RegisterVersion(concrete, bin); err != nil {
		return fmt.Errorf("register PHP %s: %w", concrete, err)
	}

	fmt.Fprintf(out, "PHP %s installed successfully.\n", label)

	return nil
}
