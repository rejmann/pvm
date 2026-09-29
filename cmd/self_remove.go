package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/cmd/selfremove"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/selfupdate"
	"github.com/rejmann/pvm/internal/shim"
	"github.com/spf13/cobra"
)

func newSelfRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "self-remove",
		Short: "Uninstall pvm from this machine",
		Long: `Remove the pvm binary and its data directory. On Windows it also removes
the pvm entries from the user PATH and the pvm block from the PowerShell
profile.

On Windows the PHP versions installed by pvm live in its data directory and
are always removed. On Linux and macOS they are system/Homebrew packages and
are kept unless --php is given.

Run it as your regular user, not with sudo. If the binary lives in a directory
you can't write to (e.g. /usr/local/bin), everything else is removed and the
command to delete the binary is printed.`,
		Example: `  pvm self-remove
  pvm self-remove --php
  pvm self-remove --yes`,
		Args: cobra.NoArgs,
		RunE: runSelfRemove,
	}
	c.Flags().BoolP("yes", "y", false, "Do not ask for confirmation")
	c.Flags().Bool("php", false, "Also remove the PHP versions installed through pvm")
	return c
}

func runSelfRemove(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	withPHP, _ := cmd.Flags().GetBool("php")

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate pvm binary: %w", err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return fmt.Errorf("locate pvm binary: %w", err)
	}

	ops := selfremove.Ops{
		RemoveVersion:     installer.New(cmd.OutOrStdout(), cmd.ErrOrStderr()).Remove,
		RemoveIntegration: shim.New(cmd.OutOrStdout(), cmd.ErrOrStderr()).RemoveIntegration,
		RemoveBinary:      selfupdate.RemoveBinary,
		Confirm: func(prompt string) bool {
			return confirm(cmd.InOrStdin(), cmd.OutOrStdout(), prompt)
		},
	}
	return selfremove.Run(home.Default(), exe, withPHP, yes, ops, cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}
