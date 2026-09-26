package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/selfupdate"
	"github.com/rejmann/pvm/internal/symlink"
	"github.com/rejmann/pvm/internal/system"
	"github.com/spf13/cobra"
)

var SelfRemoveCmd = &cobra.Command{
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

func init() {
	SelfRemoveCmd.Flags().BoolP("yes", "y", false, "Do not ask for confirmation")
	SelfRemoveCmd.Flags().Bool("php", false, "Also remove the PHP versions installed through pvm")
}

// selfRemoveOps holds the side effects of self-remove that touch the system,
// so tests can replace them.
type selfRemoveOps struct {
	removeVersion     RemoverFunc
	removeIntegration func(base, binDir string) error
	removeBinary      func(exe string) error
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

	ops := selfRemoveOps{
		removeVersion:     installer.Remove,
		removeIntegration: symlink.RemoveIntegration,
		removeBinary:      selfupdate.RemoveBinary,
	}
	return selfRemove(phpfs.NewManager(baseDir()), exe, withPHP, yes, ops,
		cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func selfRemove(m *phpfs.Manager, exe string, withPHP, yes bool, ops selfRemoveOps, in io.Reader, out, errOut io.Writer) error {
	if err := checkRemovableBase(m.Base); err != nil {
		return err
	}

	versions, err := m.InstalledVersions()
	if err != nil {
		return fmt.Errorf("list installed versions: %w", err)
	}
	// on Windows the PHP builds live in the data directory and go with it
	phpGoes := withPHP || runtime.GOOS == system.Windows

	fmt.Fprintln(out, "This will remove:")
	fmt.Fprintf(out, "  %s\n", exe)
	fmt.Fprintf(out, "  %s\n", m.Base)
	if len(versions) > 0 {
		if phpGoes {
			fmt.Fprintf(out, "  PHP %s\n", strings.Join(versions, ", "))
		} else {
			fmt.Fprintf(out, "PHP %s will be kept (pass --php to remove them too).\n", strings.Join(versions, ", "))
		}
	}

	if !yes && !confirm(in, out, "Continue? [y/N] ") {
		fmt.Fprintln(out, "Aborted.")
		return nil
	}

	if withPHP {
		for _, v := range versions {
			if err := ops.removeVersion(m.Base, v); err != nil {
				return fmt.Errorf("remove PHP %s: %w", v, err)
			}
			fmt.Fprintf(out, "PHP %s removed.\n", v)
		}
	}

	if err := ops.removeIntegration(m.Base, filepath.Dir(exe)); err != nil {
		fmt.Fprintf(errOut, "Warning: %v\n", err)
	}

	if err := os.RemoveAll(m.Base); err != nil {
		return fmt.Errorf("remove %s: %w", m.Base, err)
	}

	if err := ops.removeBinary(exe); err != nil {
		if errors.Is(err, selfupdate.ErrPermission) {
			if runtime.GOOS == system.Windows {
				return fmt.Errorf("%w — delete %s from a terminal opened as Administrator", err, exe)
			}
			return fmt.Errorf("%w — finish with: sudo rm %s", err, exe)
		}
		return err
	}

	fmt.Fprintln(out, "pvm removed.")
	if runtime.GOOS == system.Windows {
		fmt.Fprintln(out, "Open a new terminal to pick up the updated PATH.")
	} else {
		fmt.Fprintf(out, "If your shell config adds %s to PATH, remove that line.\n", symlink.ShimDir(m.Base))
	}
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

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}
