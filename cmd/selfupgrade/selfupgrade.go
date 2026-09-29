// Package selfupgrade holds the logic of pvm self-upgrade.
package selfupgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rejmann/pvm/internal/selfupdate"
)

// Run upgrades the pvm binary exe, whose version is current, to tag (the
// latest release when empty). With check it only reports what it would do.
func Run(ctx context.Context, u *selfupdate.Updater, exe, current, tag string, check bool, out io.Writer) error {
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
