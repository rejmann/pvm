package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/phpext"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/sysphp"
	"github.com/spf13/cobra"
)

func newExtCmd() *cobra.Command {
	ext := &cobra.Command{
		Use:   "ext",
		Short: "Manage the PHP extensions of an installed version",
		Long: `Manage the PHP extensions of an installed version: by default the version
in use in the current directory ($PVM_VERSION, the nearest .php-version, then
the global version), or the one given with -v/--version.

pvm installs extensions the way the version was installed: the
distribution's packages on Linux (php8.3-redis...), Homebrew formulas on macOS
(shivammathur/extensions/redis@8.3), and the DLLs bundled with the PHP build on
Windows, enabled in its php.ini. pvm remove uninstalls them with the version.

Versions installed outside pvm can be listed but are never changed.`,
	}
	ext.PersistentFlags().StringP("version", "v", "", "PHP version to work on (default: the version in use)")
	ext.AddCommand(
		&cobra.Command{
			Use:     "list [ls]",
			Aliases: []string{"ls"},
			Short:   "List the extensions PHP loads",
			Args:    cobra.NoArgs,
			RunE:    runExtList,
		},
		&cobra.Command{
			Use:     "add <extension>...",
			Short:   "Install extensions (e.g. redis, intl, xdebug)",
			Example: "  pvm ext add redis xdebug\n  pvm ext add -v 8.2 intl",
			Args:    cobra.MinimumNArgs(1),
			RunE:    runExtAdd,
		},
		&cobra.Command{
			Use:     "remove [rm] <extension>...",
			Aliases: []string{"rm"},
			Short:   "Uninstall extensions (the ones that ship with PHP are disabled instead)",
			Args:    cobra.MinimumNArgs(1),
			RunE:    runExtRemove,
		},
		&cobra.Command{
			Use:   "enable <extension>...",
			Short: "Turn installed extensions back on",
			Args:  cobra.MinimumNArgs(1),
			RunE:  func(cmd *cobra.Command, args []string) error { return runExtToggle(cmd, args, true) },
		},
		&cobra.Command{
			Use:     "disable <extension>...",
			Short:   "Turn installed extensions off without uninstalling them (e.g. xdebug)",
			Example: "  pvm ext disable xdebug",
			Args:    cobra.MinimumNArgs(1),
			RunE:    func(cmd *cobra.Command, args []string) error { return runExtToggle(cmd, args, false) },
		},
	)
	return ext
}

// extTarget wires pvm ext to this system and picks the version it works on.
func extTarget(cmd *cobra.Command, readOnly bool) (*pvm.Extensions, pvm.Active, error) {
	e := &pvm.Extensions{
		Manager:   newManager(cmd),
		Installer: installer.New(cmd.OutOrStdout(), cmd.ErrOrStderr()),
		Probe:     sysphp.Probe,
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, pvm.Active{}, err
	}
	arg, _ := cmd.Flags().GetString("version")
	a, err := e.Version(arg, dir, os.Getenv(pvm.EnvVersion), readOnly)
	return e, a, err
}

func runExtList(cmd *cobra.Command, _ []string) error {
	e, a, err := extTarget(cmd, true)
	if err != nil {
		return err
	}
	return listExtensions(e, a, cmd.OutOrStdout())
}

func listExtensions(e *pvm.Extensions, a pvm.Active, out io.Writer) error {
	exts, err := e.List(a)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "PHP %s loads:\n", a.Version)
	for _, ext := range exts {
		fmt.Fprintf(out, "  %s\n", ext)
	}
	return nil
}

func runExtAdd(cmd *cobra.Command, args []string) error {
	e, a, err := extTarget(cmd, false)
	if err != nil {
		return err
	}
	return addExtensions(e, a, args, cmd.OutOrStdout())
}

func addExtensions(e *pvm.Extensions, a pvm.Active, exts []string, out io.Writer) error {
	added, loaded, err := e.Add(a, exts)
	if len(loaded) > 0 {
		fmt.Fprintf(out, "PHP %s already loads %s.\n", a.Version, strings.Join(loaded, ", "))
	}
	if err != nil {
		return err
	}
	if len(added) > 0 {
		fmt.Fprintf(out, "Installed %s for PHP %s.\n", strings.Join(added, ", "), a.Version)
	}
	return nil
}

func runExtRemove(cmd *cobra.Command, args []string) error {
	e, a, err := extTarget(cmd, false)
	if err != nil {
		return err
	}
	r, err := e.Remove(a, args)
	printRemoval(cmd.OutOrStdout(), r, a.Version)
	return err
}

func printRemoval(out io.Writer, r phpext.Removal, version string) {
	if len(r.Uninstalled) > 0 {
		fmt.Fprintf(out, "Removed %s from PHP %s.\n", strings.Join(r.Uninstalled, ", "), version)
	}
	for _, ext := range r.Disabled {
		fmt.Fprintf(out, "%s ships with PHP %s (pvm did not install it), so it was disabled instead of uninstalled. "+
			"Turn it back on with: pvm ext enable %s\n", ext, version, ext)
	}
}

func runExtToggle(cmd *cobra.Command, args []string, enabled bool) error {
	e, a, err := extTarget(cmd, false)
	if err != nil {
		return err
	}
	if err := e.SetEnabled(a, args, enabled); err != nil {
		return err
	}
	done := "Disabled"
	if enabled {
		done = "Enabled"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s for PHP %s.\n", done, strings.Join(args, ", "), a.Version)
	return nil
}
