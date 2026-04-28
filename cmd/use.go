package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/version"
	"github.com/rejmann/pvm/system"
	"github.com/spf13/cobra"
)

var UseCmd = &cobra.Command{
	Use:     "use [u] <version|lts>",
	Aliases: []string{"u"},
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
	printPathHint(out, m.Base)
	return nil
}

func printPathHint(out io.Writer, base string) {
	var managed string
	switch runtime.GOOS {
	case system.Linux:
		managed = filepath.Join(base, "bin")
	default:
		managed = filepath.Join(base, "shims")
	}

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
