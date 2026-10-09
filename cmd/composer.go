package cmd

import (
	"os"

	composercli "github.com/rejmann/pvm/cmd/composer"
	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/progress"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/sysphp"
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

Before install and update, pvm checks the ext-* requirements of composer.json
(and, for install, composer.lock) against the PHP version and offers to install
the missing ones. If Composer still stops because the version lacks an
extension (ext-xml, ext-intl...), pvm offers to install it and runs the same
command again. Likewise, if the zip extension is missing and
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
}

func runComposer(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	m := newManager(cmd)
	source := composer.New()
	source.Progress = progress.Terminal(cmd.ErrOrStderr())

	c := &pvm.Composer{
		Manager:    m,
		Source:     source,
		Extensions: installer.New(cmd.OutOrStdout(), cmd.ErrOrStderr()),
		Probe:      sysphp.Probe,
		Exec:       process.Run,
		Getenv:     os.Getenv,
		CanUnzip:   composercli.CanUnzip(os.Getenv("PATH")),
		Notices:    cmd.ErrOrStderr(),
	}
	if composercli.IsTerminal(os.Stdin) {
		c.Confirm = func(prompt string) bool { return confirm(cmd.Context(), os.Stdin, cmd.ErrOrStderr(), prompt) }
	}

	if composercli.ForceANSI(args, composercli.IsTerminal(os.Stdout), os.Getenv("NO_COLOR")) {
		args = append([]string{"--ansi"}, args...)
	}
	code, err := c.Run(cmd.Context(), dir, args)
	if err != nil {
		return err
	}
	os.Exit(code)
	return nil
}
