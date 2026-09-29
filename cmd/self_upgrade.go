package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rejmann/pvm/cmd/selfupgrade"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/selfupdate"
	"github.com/spf13/cobra"
)

func newSelfUpgradeCmd(version string) *cobra.Command {
	c := &cobra.Command{
		Use:   "self-upgrade [tag]",
		Short: "Upgrade pvm itself to the latest release (or to the given tag)",
		Long: `Download a pvm release from GitHub and replace the running binary with it.

Without arguments it installs the latest release; pass a tag (e.g. v1.2.0) to
install that one instead, which also allows downgrading. Use --check to only
report whether a newer release exists.

It never needs sudo. pvm installed as the README shows (~/.pvm/bin on Linux
and macOS) is replaced in place. An older install in a root-owned directory
such as /usr/local/bin is left alone: the new pvm goes into ~/.pvm/bin
instead, the php shim is pointed at it, and pvm tells you if that directory
must come first in your PATH.`,
		Example: `  pvm self-upgrade
  pvm self-upgrade --check
  pvm self-upgrade v1.2.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpgrade(cmd, args, version)
		},
	}
	c.Flags().BoolP("check", "c", false, "Only check whether a newer release is available")
	return c
}

// runSelfUpgrade upgrades the running pvm, whose version is current.
func runSelfUpgrade(cmd *cobra.Command, args []string, current string) error {
	check, _ := cmd.Flags().GetBool("check")

	var tag string
	if len(args) == 1 {
		tag = args[0]
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate pvm binary: %w", err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return fmt.Errorf("locate pvm binary: %w", err)
	}

	h := home.Default()
	out := cmd.OutOrStdout()
	now, err := selfupgrade.Run(cmd.Context(), selfupdate.New(), exe, current, tag, check, selfupgrade.FallbackDir(h), out)
	if err != nil || now == exe {
		return err
	}
	return selfupgrade.Moved(h, exe, now, out)
}
