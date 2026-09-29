package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

If pvm lives in a directory you can't write to (e.g. /usr/local/bin), run it
with sudo (Windows: from a terminal opened as Administrator).`,
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

	return selfUpgrade(cmd.Context(), selfupdate.New(), exe, current, tag, check, cmd.OutOrStdout())
}

func selfUpgrade(ctx context.Context, u *selfupdate.Updater, exe, current, tag string, check bool, out io.Writer) error {
	selfupdate.RemoveOld(exe)

	if tag == "" {
		latest, err := u.LatestTag(ctx)
		if err != nil {
			return err
		}
		tag = latest
	} else if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	if selfupdate.SameVersion(current, tag) {
		fmt.Fprintf(out, "pvm is already at %s.\n", tag)
		return nil
	}

	if check {
		fmt.Fprintf(out, "pvm %s is available (current: %s). Run: pvm self-upgrade\n", tag, current)
		return nil
	}

	fmt.Fprintf(out, "Downloading pvm %s...\n", tag)
	bin, err := u.Download(ctx, tag)
	if err != nil {
		return err
	}

	if err := selfupdate.Replace(exe, bin); err != nil {
		if errors.Is(err, selfupdate.ErrPermission) {
			return fmt.Errorf("%w — %s", err, elevatedHint("pvm self-upgrade"))
		}
		return err
	}

	fmt.Fprintf(out, "pvm upgraded from %s to %s (%s).\n", current, tag, exe)
	return nil
}
