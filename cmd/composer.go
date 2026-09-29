package cmd

import (
	"os"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/sysphp"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var ComposerCmd = &cobra.Command{
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

If Composer stops because the PHP version lacks an extension the project
needs (ext-xml, ext-intl...), pvm offers to install it for that version and
runs the same command again. Likewise, if the zip extension is missing and
neither unzip nor 7z is available, pvm offers to install it up front.

PVM_VERSION is set for the process, so scripts Composer runs that call php
use the same version.`,
	Example: `  pvm composer install
  pvm composer require monolog/monolog
  pvm composer -V
  PVM_VERSION=8.2 pvm composer update`,
	DisableFlagParsing: true,
	RunE:               runComposer,
}

func runComposer(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	m := newManager(cmd)

	c := &pvm.Composer{
		Manager:    m,
		Source:     composer.New(),
		Extensions: installer.New(cmd.OutOrStdout(), cmd.ErrOrStderr()),
		Probe:      sysphp.Probe,
		Exec:       process.Run,
		Getenv:     os.Getenv,
		CanUnzip:   canUnzip(os.Getenv("PATH")),
		Notices:    cmd.ErrOrStderr(),
	}
	if isTerminal(os.Stdin) {
		c.Confirm = func(prompt string) bool { return confirm(os.Stdin, cmd.ErrOrStderr(), prompt) }
	}

	if forceANSI(args, isTerminal(os.Stdout), os.Getenv("NO_COLOR")) {
		args = append([]string{"--ansi"}, args...)
	}
	code, err := c.Run(cmd.Context(), dir, args)
	if err != nil {
		return err
	}
	os.Exit(code)
	return nil
}

// canUnzip reports whether Composer can extract zips without the PHP
// extension: it also accepts unzip or 7-Zip on PATH.
func canUnzip(path string) bool {
	for _, name := range []string{"unzip", "7z", "7zz"} {
		if process.LookPath(name, path, "") != "" {
			return true
		}
	}
	return false
}

// forceANSI reports whether to pass --ansi: Composer's stderr goes through
// pvm (to spot missing extensions), so Composer would otherwise turn colors
// off even in a terminal.
func forceANSI(args []string, stdoutTerminal bool, noColor string) bool {
	if !stdoutTerminal || noColor != "" {
		return false
	}
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "--ansi" || a == "--no-ansi" {
			return false
		}
	}
	return true
}

// isTerminal is false for pipes and redirects, including </dev/null (a char
// device, so a mode check alone is not enough).
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
