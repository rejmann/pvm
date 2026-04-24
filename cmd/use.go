package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	printPathHint(m.Base, out)
	return nil
}

func printPathHint(base string, out io.Writer) {
	pvmBin := filepath.Join(base, "bin")
	entries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	inPath := false
	isFirst := false
	for i, e := range entries {
		if e == pvmBin {
			inPath = true
			isFirst = (i == 0)
			break
		}
	}

	if !inPath {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "⚠  pvm is not in your PATH — 'php' still points to the system binary.")
		fmt.Fprintln(out, "   Run this now to activate it in your current shell:")
		fmt.Fprintf(out, "     export PATH=\"%s:$PATH\"\n", pvmBin)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "   To make it permanent, add to ~/.zshrc or ~/.bashrc:")
		fmt.Fprintf(out, "     export PATH=\"%s:$PATH\"\n", pvmBin)
		fmt.Fprintln(out, "   Then restart your shell or run: source ~/.zshrc")
	} else if !isFirst {
		fmt.Fprintf(out, "\n⚠  %s is in PATH but not first — system PHP may shadow it.\n", pvmBin)
	}
}
