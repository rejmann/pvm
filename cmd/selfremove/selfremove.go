// Package selfremove holds the logic of pvm self-remove.
package selfremove

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/selfupdate"
)

// Ops holds the side effects of self-remove that touch the system or the
// user, so tests can replace them.
type Ops struct {
	RemoveVersion     func(h *home.Dir, ver string) error
	RemoveIntegration func(h *home.Dir, binDir string) error
	RemoveBinary      func(exe string) error
	Confirm           func(prompt string) bool // asks the user yes/no
}

// Run removes the pvm binary exe and the data directory h — and, with
// withPHP, the PHP versions pvm installed — after confirming unless yes.
func Run(h *home.Dir, exe string, withPHP, yes bool, ops Ops, out, errOut io.Writer) error {
	if err := checkRemovableBase(h.Path); err != nil {
		return err
	}

	versions, err := h.Versions()
	if err != nil {
		return fmt.Errorf("list installed versions: %w", err)
	}
	// on Windows the PHP builds live in the data directory and go with it
	phpGoes := withPHP || phpInHome

	fmt.Fprintln(out, "This will remove:")
	fmt.Fprintf(out, "  %s\n", exe)
	fmt.Fprintf(out, "  %s\n", h.Path)
	if len(versions) > 0 {
		if phpGoes {
			fmt.Fprintf(out, "  PHP %s\n", strings.Join(versions, ", "))
		} else {
			fmt.Fprintf(out, "PHP %s will be kept (pass --php to remove them too).\n", strings.Join(versions, ", "))
		}
	}

	if !yes && !ops.Confirm("Continue? [y/N] ") {
		fmt.Fprintln(out, "Aborted.")
		return nil
	}

	if withPHP {
		for _, v := range versions {
			if err := ops.RemoveVersion(h, v); err != nil {
				return fmt.Errorf("remove PHP %s: %w", v, err)
			}
			fmt.Fprintf(out, "PHP %s removed.\n", v)
		}
	}

	if err := ops.RemoveIntegration(h, filepath.Dir(exe)); err != nil {
		fmt.Fprintf(errOut, "Warning: %v\n", err)
	}

	if err := os.RemoveAll(h.Path); err != nil {
		return fmt.Errorf("remove %s: %w", h.Path, err)
	}

	if err := ops.RemoveBinary(exe); err != nil {
		if errors.Is(err, selfupdate.ErrPermission) {
			return fmt.Errorf("%w — %s", err, removeBinaryHint(exe))
		}
		return err
	}

	fmt.Fprintln(out, "pvm removed.")
	printAfterRemoval(out, h.ShimDir())
	return nil
}

// checkRemovableBase guards against deleting a directory that is not pvm's
// own, e.g. when PVM_HOME points at the home or root directory.
func checkRemovableBase(base string) error {
	clean := filepath.Clean(base)
	home, _ := os.UserHomeDir()
	if !filepath.IsAbs(clean) || filepath.Dir(clean) == clean || (home != "" && clean == filepath.Clean(home)) {
		return fmt.Errorf("refusing to remove %s: the pvm data directory (PVM_HOME) must be a dedicated directory", base)
	}
	return nil
}
