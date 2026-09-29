package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/installer"
	"github.com/rejmann/pvm/internal/pvm"
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
	h := m.Home

	a, err := m.Active(dir, os.Getenv(pvm.EnvVersion))
	if errors.Is(err, pvm.ErrNoActiveVersion) {
		return fmt.Errorf("%w — run: pvm use <version>", err)
	}
	if err != nil {
		return err
	}

	info, err := probePHP(a.Binary)
	if err != nil {
		return err
	}
	interactive := isTerminal(os.Stdin)
	if !info.Zip && !hasArchiveTool(os.Getenv("PATH")) {
		offerExtensions(h, a.Version, []string{"zip"}, "which Composer needs to extract packages",
			interactive, os.Stdin, os.Stderr)
	}

	phar, err := composer.New().Ensure(cmd.Context(), h.ComposerDir(), a.Version, info.Version, func(r composer.Release) {
		fmt.Fprintf(os.Stderr, "Downloading Composer %s for PHP %s...\n", r.Version, a.Version)
	})
	if err != nil {
		return err
	}

	env := composer.Env(h.ComposerDir(), a.Version, os.Getenv)
	env[pvm.EnvVersion] = a.Version
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}

	phpArgs := append([]string{phar}, args...)
	if forceANSI(args, isTerminal(os.Stdout), os.Getenv("NO_COLOR")) {
		phpArgs = append([]string{phar, "--ansi"}, args...)
	}

	code, out, err := runChild(a.Binary, phpArgs)
	if err != nil {
		return err
	}
	// Composer may exit 0 even so: Symfony Flex reports a failed update after
	// create-project without failing the command.
	if len(out.missing) > 0 &&
		offerExtensions(h, a.Version, out.missing, "which this project needs", interactive, os.Stdin, os.Stderr) {
		// Composer only creates a project in a new or empty directory, so
		// emptying it restores the state the command started from.
		if out.project != "" {
			if err := emptyDir(out.project); err != nil {
				return fmt.Errorf("clean up %s before running Composer again: %w", out.project, err)
			}
		}
		fmt.Fprintln(os.Stderr, "Running Composer again...")
		if code, _, err = runChild(a.Binary, phpArgs); err != nil {
			return err
		}
	}
	os.Exit(code)
	return nil
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

// offerExtensions asks to install exts for version and reports whether they
// were installed. It never fails the command: at worst Composer fails as it
// would have without pvm.
func offerExtensions(h *home.Dir, version string, exts []string, why string, interactive bool, in io.Reader, out io.Writer) bool {
	what := "the " + exts[0] + " extension"
	if len(exts) > 1 {
		what = "the " + strings.Join(exts, ", ") + " extensions"
	}
	fmt.Fprintf(out, "PHP %s is missing %s, %s.\n", version, what, why)
	if !interactive {
		fmt.Fprintln(out, "Run pvm composer in a terminal to let pvm install it.")
		return false
	}
	if !confirm(in, out, "Install now? [y/N] ") {
		return false
	}
	if err := installExtensions(h, version, exts); err != nil {
		fmt.Fprintf(out, "Warning: %v\n", err)
		return false
	}
	fmt.Fprintf(out, "Installed %s for PHP %s.\n", what, version)
	return true
}

// installExtensions installs PHP extensions; tests replace it.
var installExtensions = func(h *home.Dir, ver string, exts []string) error {
	return installer.New(os.Stdout, os.Stderr).AddExtensions(h, ver, exts)
}

// missingExtRe matches Composer's platform errors, e.g.
// "symfony/framework-bundle[v8.1.0, ..., v8.1.7] require ext-xml * -> it is missing from your system."
// "Root composer.json requires PHP extension ext-intl * but it is missing from your system."
var (
	missingExtRe = regexp.MustCompile(`\bext-([A-Za-z0-9_]+)\b.*missing from your system`)
	projectRe    = regexp.MustCompile(`^Created project in (.+?)\s*$`)
	ansiRe       = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")
)

// composerOutput watches Composer's stderr line by line for what pvm acts on.
type composerOutput struct {
	missing []string // extensions reported missing, in order, without repeats
	project string   // directory create-project created, if any
	partial []byte   // unfinished last line
}

// maxLine bounds a line pvm holds on to; the ones it looks for are short.
const maxLine = 64 << 10

func (o *composerOutput) Write(p []byte) (int, error) {
	o.partial = append(o.partial, p...)
	for {
		i := bytes.IndexByte(o.partial, '\n')
		if i < 0 {
			break
		}
		o.line(string(o.partial[:i]))
		o.partial = o.partial[i+1:]
	}
	if len(o.partial) > maxLine {
		o.partial = o.partial[len(o.partial)-maxLine:]
	}
	return len(p), nil
}

// Close handles a last line without a newline.
func (o *composerOutput) Close() {
	if len(o.partial) > 0 {
		o.line(string(o.partial))
		o.partial = nil
	}
}

func (o *composerOutput) line(l string) {
	l = strings.TrimRight(ansiRe.ReplaceAllString(l, ""), "\r")
	if m := projectRe.FindStringSubmatch(l); m != nil {
		o.project = m[1]
	}
	for _, m := range missingExtRe.FindAllStringSubmatch(l, -1) {
		ext := strings.ToLower(m[1])
		if !slices.Contains(o.missing, ext) {
			o.missing = append(o.missing, ext)
		}
	}
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

// runChild runs bin with inherited stdin/stdout and stderr copied to pvm's
// stderr, returning its exit code and what its stderr reported. Ctrl+C
// reaches the child directly (same process group), so pvm only ignores it;
// other termination signals are forwarded.
func runChild(bin string, args []string) (code int, out *composerOutput, err error) {
	out = &composerOutput{}
	c := exec.Command(bin, args...)
	c.Stdin, c.Stdout = os.Stdin, os.Stdout
	c.Stderr = io.MultiWriter(os.Stderr, out)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(sig)
		close(sig)
	}()

	if err := c.Start(); err != nil {
		return 0, nil, fmt.Errorf("run %s: %w", bin, err)
	}
	go func() {
		for s := range sig {
			if s != os.Interrupt {
				_ = c.Process.Signal(s)
			}
		}
	}()

	err = c.Wait()
	out.Close()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
		if code < 0 { // killed by a signal
			code = 1
		}
	case err != nil:
		return 0, nil, fmt.Errorf("run %s: %w", bin, err)
	}
	return code, out, nil
}

// isTerminal is false for pipes and redirects, including </dev/null (a char
// device, so a mode check alone is not enough).
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
