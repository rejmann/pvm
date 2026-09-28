package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rejmann/pvm/internal/composer"
	phpfs "github.com/rejmann/pvm/internal/fs"
	"github.com/rejmann/pvm/internal/installer"
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

func runComposer(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	base := baseDir()

	a, err := resolveActive(phpfs.NewManager(base), dir, os.Getenv(envVersion))
	if errors.Is(err, ErrNoActiveVersion) {
		return fmt.Errorf("%w — run: pvm use <version>", err)
	}
	if err != nil {
		return err
	}

	info, err := probePHP(a.Binary)
	if err != nil {
		return err
	}
	if !info.Zip && !hasArchiveTool(os.Getenv("PATH")) {
		offerZipExtension(base, a.Version, isTerminal(os.Stdin), os.Stdin, os.Stderr)
	}

	phar, err := composer.New().Ensure(cmd.Context(), base, a.Version, info.Version, func(r composer.Release) {
		fmt.Fprintf(os.Stderr, "Downloading Composer %s for PHP %s...\n", r.Version, a.Version)
	})
	if err != nil {
		return err
	}

	env := composer.Env(base, a.Version, os.Getenv)
	env[envVersion] = a.Version
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return execBinary(a.Binary, append([]string{phar}, args...))
}

type phpInfo struct {
	Version string // exact version, e.g. "8.3.12"
	Zip     bool   // zip extension loaded
}

// probeScript prints the version and whether zip is loaded on the last two
// lines, so startup warnings printed before them are ignored.
const probeScript = `echo "\n", PHP_MAJOR_VERSION, ".", PHP_MINOR_VERSION, ".", PHP_RELEASE_VERSION, "\n", extension_loaded("zip") ? 1 : 0;`

// probePHP asks the php binary for its exact version, since pvm may only know
// the branch (8.3) and Composer's minimum PHP is a patch (7.2.5), and whether
// it can extract zip archives. php.ini is loaded, as Composer will load it.
func probePHP(bin string) (phpInfo, error) {
	out, err := exec.Command(bin, "-r", probeScript).Output()
	if err != nil {
		return phpInfo{}, fmt.Errorf("run %s: %w", bin, err)
	}
	return parseProbe(string(out))
}

func parseProbe(out string) (phpInfo, error) {
	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	if len(lines) < 2 {
		return phpInfo{}, fmt.Errorf("unexpected php output: %q", out)
	}
	return phpInfo{
		Version: strings.TrimSpace(lines[len(lines)-2]),
		Zip:     strings.TrimSpace(lines[len(lines)-1]) == "1",
	}, nil
}

// hasArchiveTool reports whether Composer can extract zips without the PHP
// extension: it also accepts unzip or 7-Zip on PATH.
func hasArchiveTool(path string) bool {
	for _, name := range []string{"unzip", "7z", "7zz"} {
		if lookPathExcluding(name, path, "") != "" {
			return true
		}
	}
	return false
}

// offerZipExtension asks to install the zip extension for version. It never
// fails the command: Composer still runs, and only package extraction needs zip.
func offerZipExtension(base, version string, interactive bool, in io.Reader, out io.Writer) {
	fmt.Fprintf(out, "PHP %s has no zip extension, which Composer needs to extract packages.\n", version)
	if !interactive {
		fmt.Fprintln(out, "Run pvm composer in a terminal to let pvm install it.")
		return
	}
	if !confirm(in, out, "Install it now? [y/N] ") {
		return
	}
	if err := installer.EnsureExtensions(base, version); err != nil {
		fmt.Fprintf(out, "Warning: %v\n", err)
		return
	}
	fmt.Fprintf(out, "zip extension installed for PHP %s.\n", version)
}

// isTerminal is false for pipes and redirects, including </dev/null (a char
// device, so a mode check alone is not enough).
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
