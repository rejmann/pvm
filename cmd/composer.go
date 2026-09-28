package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/php"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/resolve"
	"github.com/spf13/cobra"
)

func newComposerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "composer [args...]",
		Short: "Run Composer with the PHP version in use (downloads composer.phar on first use)",
		Long: `Run Composer with the pvm-managed PHP version in use in the current
directory ($PVM_VERSION, the nearest .php-version, then the global version).
Every argument is passed to Composer unchanged, including -h, -V and --.

Nothing is installed globally. Each PHP version gets its own composer.phar —
the newest Composer that supports it, downloaded from getcomposer.org the first
time it is needed and verified with Composer's signing key — and its own
Composer home (config, auth.json, global packages), all in the pvm home. So
self-update, --rollback or global require for one version never affect another.
Only the download cache is shared. COMPOSER_HOME / COMPOSER_CACHE_DIR, if set,
are respected. pvm remove deletes a version's Composer with it.

If the PHP version lacks the zip extension and neither unzip nor 7z is
available, pvm offers to install the extension for that version.

PVM_VERSION is set for the process, so scripts Composer runs that call php
use the same version.`,
		Example: `  pvm composer install
  pvm composer require monolog/monolog
  pvm composer -V
  PVM_VERSION=8.2 pvm composer update`,
		DisableFlagParsing: true,
		RunE:               runComposer,
	}
}

func runComposer(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	h := home.Default()

	a, err := resolve.Active(h, dir, os.Getenv(resolve.EnvVersion))
	if errors.Is(err, resolve.ErrNoActiveVersion) {
		return fmt.Errorf("%w — run: pvm use <version>", err)
	}
	if err != nil {
		return err
	}

	info, err := php.Probe(a.Binary)
	if err != nil {
		return err
	}
	if !info.Zip && !hasArchiveTool(os.Getenv("PATH")) {
		offerZipExtension(h, a.Version, isTerminal(os.Stdin), os.Stdin, os.Stderr)
	}

	phar, err := composer.New().Ensure(cmd.Context(), h.ComposerDir(), a.Version, info.Version, func(r composer.Release) {
		fmt.Fprintf(os.Stderr, "Downloading Composer %s for PHP %s...\n", r.Version, a.Version)
	})
	if err != nil {
		return err
	}

	env := composer.Env(h.ComposerDir(), a.Version, os.Getenv)
	env[resolve.EnvVersion] = a.Version
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return proc.Exec(a.Binary, append([]string{phar}, args...))
}

// hasArchiveTool reports whether Composer can extract zips without the PHP
// extension: it also accepts unzip or 7-Zip on PATH.
func hasArchiveTool(path string) bool {
	for _, name := range []string{"unzip", "7z", "7zz"} {
		if proc.LookPathExcluding(name, path, "") != "" {
			return true
		}
	}
	return false
}

// offerZipExtension asks to install the zip extension for version. It never
// fails the command: Composer still runs, and only package extraction needs zip.
func offerZipExtension(h *home.Dir, version string, interactive bool, in io.Reader, out io.Writer) {
	fmt.Fprintf(out, "PHP %s has no zip extension, which Composer needs to extract packages.\n", version)
	if !interactive {
		fmt.Fprintln(out, "Run pvm composer in a terminal to let pvm install it.")
		return
	}
	if !confirm(in, out, "Install it now? [y/N] ") {
		return
	}
	if err := installer.EnsureExtensions(h, version, proc.Streams{Out: out, Err: out}); err != nil {
		fmt.Fprintf(out, "Warning: %v\n", err)
		return
	}
	fmt.Fprintf(out, "zip extension installed for PHP %s.\n", version)
}
