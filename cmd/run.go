package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/rejmann/pvm/cmd/run"
	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
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
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		return cmd.Help()
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	versionArg, rest, err := run.SplitArgs(args)
	if err != nil {
		return err
	}
	if err := run.CheckFile(rest, dir); err != nil {
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
