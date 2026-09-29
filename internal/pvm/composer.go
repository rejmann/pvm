package pvm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/sysphp"
)

// ComposerSource provides the composer.phar of a PHP version.
type ComposerSource interface {
	Ensure(ctx context.Context, root, phpVersion, phpExact string, onDownload func(composer.Release)) (string, error)
}

// ExtensionInstaller adds PHP extensions to an installed version.
type ExtensionInstaller interface {
	AddExtensions(h *home.Dir, ver string, exts []string) error
}

// Composer runs Composer with the PHP version in use and offers to install
// the PHP extensions it reports missing.
type Composer struct {
	Manager    *Manager
	Source     ComposerSource
	Extensions ExtensionInstaller

	// Probe inspects a php binary (sysphp.Probe).
	Probe func(bin string) (sysphp.Info, error)
	// Exec runs a program with extra env vars, copying its stderr to watch,
	// and returns its exit code (process.Run).
	Exec func(bin string, args []string, env map[string]string, watch io.Writer) (int, error)
	// Confirm asks the user a yes/no question; nil when there is no terminal.
	Confirm func(prompt string) bool
	Getenv  func(string) string

	CanUnzip bool      // unzip or 7z is available, so Composer does not need ext-zip
	Notices  io.Writer // pvm's own messages
}

// Run runs Composer with args for the version in use in dir and returns its
// exit code.
func (c *Composer) Run(ctx context.Context, dir string, args []string) (int, error) {
	h := c.Manager.Home
	a, err := c.Manager.Active(dir, c.Getenv(EnvVersion))
	if errors.Is(err, ErrNoActiveVersion) {
		return 0, fmt.Errorf("%w — run: pvm use <version>", err)
	}
	if err != nil {
		return 0, err
	}

	info, err := c.Probe(a.Binary)
	if err != nil {
		return 0, err
	}
	if !info.Zip && !c.CanUnzip {
		c.offerExtensions(a.Version, []string{"zip"}, "which Composer needs to extract packages")
	}

	phar, err := c.Source.Ensure(ctx, h.ComposerDir(), a.Version, info.Version, func(r composer.Release) {
		fmt.Fprintf(c.Notices, "Downloading Composer %s for PHP %s...\n", r.Version, a.Version)
	})
	if err != nil {
		return 0, err
	}

	env := composer.Env(h.ComposerDir(), a.Version, c.Getenv)
	env[EnvVersion] = a.Version
	phpArgs := append([]string{phar}, args...)

	out := &composer.Output{}
	code, err := c.Exec(a.Binary, phpArgs, env, out)
	if err != nil {
		return 0, err
	}
	out.Flush()

	// Composer may exit 0 even so: Symfony Flex reports a failed update after
	// create-project without failing the command.
	if len(out.Missing) == 0 || !c.offerExtensions(a.Version, out.Missing, "which this project needs") {
		return code, nil
	}
	// Composer only creates a project in a new or empty directory, so
	// emptying it restores the state the command started from.
	if out.Project != "" {
		if err := emptyDir(out.Project); err != nil {
			return 0, fmt.Errorf("clean up %s before running Composer again: %w", out.Project, err)
		}
	}
	fmt.Fprintln(c.Notices, "Running Composer again...")
	return c.Exec(a.Binary, phpArgs, env, io.Discard)
}

// offerExtensions asks to install exts for version and reports whether they
// were installed. It never fails the command: at worst Composer fails as it
// would have without pvm.
func (c *Composer) offerExtensions(version string, exts []string, why string) bool {
	what := "the " + exts[0] + " extension"
	if len(exts) > 1 {
		what = "the " + strings.Join(exts, ", ") + " extensions"
	}
	fmt.Fprintf(c.Notices, "PHP %s is missing %s, %s.\n", version, what, why)
	if c.Confirm == nil {
		fmt.Fprintln(c.Notices, "Run pvm composer in a terminal to let pvm install it.")
		return false
	}
	if !c.Confirm("Install now? [y/N] ") {
		return false
	}
	if err := c.Extensions.AddExtensions(c.Manager.Home, version, exts); err != nil {
		fmt.Fprintf(c.Notices, "Warning: %v\n", err)
		return false
	}
	fmt.Fprintf(c.Notices, "Installed %s for PHP %s.\n", what, version)
	return true
}

// emptyDir removes everything inside dir, keeping dir itself (it may be the
// working directory).
func emptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
