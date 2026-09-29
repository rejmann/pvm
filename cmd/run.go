package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/rejmann/pvm/internal/version"
	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:   "run [-v version | version] <file> [args...]",
	Short: "Run a PHP file with a specific installed version (default: the version in use)",
	Long: `Run a PHP file with a specific installed version, without changing the
global or project version. The file can be given directly or with -f/--file;
the arguments after it are passed to the script.

The version is optional and can be given as the first argument or with
-v/--version. It must be installed. Without it, the file runs with the version
in use in the current directory ($PVM_VERSION, the nearest .php-version, then
the global version).

PVM_VERSION is set for the php process, so tools it starts that call php
(e.g. Composer or scripts with #!/usr/bin/env php) use the same version.`,
	Example: `  pvm run 8.5 script.php
  pvm run 8.2 --file script.php arg1 arg2
  pvm run --version 8.2 script.php
  pvm run -v lts script.php
  pvm run lts script.php
  pvm run script.php       # version in use`,
	DisableFlagParsing: true,
	RunE:               runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		return cmd.Help()
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	versionArg, rest, err := splitRunArgs(args)
	if err != nil {
		return err
	}
	if err := checkRunFile(rest, dir); err != nil {
		return err
	}

	a, err := newManager(cmd).Select(versionArg, dir, os.Getenv(pvm.EnvVersion))
	if errors.Is(err, pvm.ErrNoActiveVersion) {
		return fmt.Errorf("%w — pass one (pvm run 8.3 ...) or run: pvm use <version>", err)
	}
	if err != nil {
		return err
	}

	if err := os.Setenv(pvm.EnvVersion, a.Version); err != nil {
		return err
	}
	return process.Exec(a.Binary, rest)
}

// splitRunArgs separates the optional version from the php arguments. The
// version comes from -v/--version anywhere before a "--", or positionally from
// a first argument that parses as a version or alias ("lts"). Everything after
// "--" goes to the script untouched.
func splitRunArgs(args []string) (versionArg string, rest []string, err error) {
	set := func(v string) error {
		if versionArg != "" {
			return fmt.Errorf("version given twice (%s and %s)", versionArg, v)
		}
		versionArg = v
		return nil
	}

	var passthrough []string
scan:
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			passthrough = args[i+1:]
			break scan
		case a == "-v" || a == "--version":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag %s needs a version", a)
			}
			if err := set(args[i+1]); err != nil {
				return "", nil, err
			}
			i++
		case strings.HasPrefix(a, "--version="):
			if err := set(strings.TrimPrefix(a, "--version=")); err != nil {
				return "", nil, err
			}
		default:
			rest = append(rest, a)
		}
	}

	if len(rest) > 0 && looksLikeVersion(rest[0]) {
		if err := set(rest[0]); err != nil {
			return "", nil, err
		}
		rest = rest[1:]
	}
	return versionArg, append(rest, passthrough...), nil
}

func looksLikeVersion(s string) bool {
	_, err := version.Parse(s)
	return err == nil || version.IsAlias(s)
}

var ErrNoRunFile = errors.New("no PHP file given — usage: pvm run [-v version | version] <file> [args...]")

// checkRunFile makes sure the php arguments start with an existing file,
// given directly or with -f/--file, so run only ever runs a script.
func checkRunFile(rest []string, dir string) error {
	if len(rest) == 0 {
		return ErrNoRunFile
	}

	file := rest[0]
	if file == "-f" || file == "--file" {
		if len(rest) < 2 {
			return ErrNoRunFile
		}
		file = rest[1]
	} else if strings.HasPrefix(file, "-") {
		return ErrNoRunFile
	}

	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("PHP file %s not found", file)
	}
	if fi.IsDir() {
		return fmt.Errorf("%s is a directory, not a PHP file", file)
	}
	return nil
}
