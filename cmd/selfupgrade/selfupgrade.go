// Package selfupgrade holds the logic of pvm self-upgrade.
package selfupgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/selfupdate"
)

// Run upgrades the pvm binary exe, whose version is current, to tag (the
// latest release when empty). With check it only reports what it would do.
// When exe's directory is not writable and fallbackDir is set, the new
// binary goes there instead, so no sudo is needed. It returns where pvm now
// is: exe, or the binary in fallbackDir.
func Run(ctx context.Context, u *selfupdate.Updater, exe, current, tag string, check bool, fallbackDir string, out io.Writer) (string, error) {
	selfupdate.RemoveOld(exe)

	if tag == "" {
		latest, err := u.LatestTag(ctx)
		if err != nil {
			return exe, err
		}
		tag = latest
	} else if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	if selfupdate.SameVersion(current, tag) {
		fmt.Fprintf(out, "pvm is already at %s.\n", tag)
		return exe, nil
	}

	if check {
		fmt.Fprintf(out, "pvm %s is available (current: %s). Run: pvm self-upgrade\n", tag, current)
		return exe, nil
	}

	fmt.Fprintf(out, "Downloading pvm %s...\n", tag)
	bin, err := u.Download(ctx, tag)
	if err != nil {
		return exe, err
	}

	target := exe
	err = selfupdate.Replace(exe, bin)
	if errors.Is(err, selfupdate.ErrPermission) && fallbackDir != "" && filepath.Dir(exe) != fallbackDir {
		target = filepath.Join(fallbackDir, filepath.Base(exe))
		if err = os.MkdirAll(fallbackDir, 0755); err == nil {
			err = selfupdate.Replace(target, bin)
		}
		if err == nil {
			fmt.Fprintf(out, "%s is not writable, so pvm is now installed in %s.\n", filepath.Dir(exe), fallbackDir)
		}
	}
	if err != nil {
		if errors.Is(err, selfupdate.ErrPermission) {
			return exe, fmt.Errorf("%w — %s", err, elevatedHint("pvm self-upgrade"))
		}
		return exe, err
	}

	fmt.Fprintf(out, "pvm upgraded from %s to %s (%s).\n", current, tag, target)
	return target, nil
}
