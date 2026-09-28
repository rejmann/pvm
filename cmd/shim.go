package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/proc"
	"github.com/rejmann/pvm/internal/resolve"
	"github.com/spf13/cobra"
)

// newShimCmd is invoked by the php shim script; it is not meant to be run by hand.
func newShimCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "shim php [args...]",
		Short:              "Run php with the version selected for the current directory",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE:               runShim,
	}
}

func runShim(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] != "php" {
		return errors.New("usage: pvm shim php [args...]")
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	bin, err := shimTarget(home.Default(), dir, os.Getenv(resolve.EnvVersion), os.Getenv("PATH"))
	if err != nil {
		return err
	}
	return proc.Exec(bin, args[1:])
}

// shimTarget returns the php binary the shim should run. When no version is
// selected anywhere, it falls back to the first php on PATH outside pvm.
func shimTarget(h *home.Dir, dir, env, path string) (string, error) {
	a, err := resolve.Active(h, dir, env)
	if err == nil {
		return a.Binary, nil
	}
	if !errors.Is(err, resolve.ErrNoActiveVersion) {
		return "", err
	}

	if bin := proc.LookPathExcluding("php", path, h.ShimDir()); bin != "" {
		return bin, nil
	}
	return "", fmt.Errorf("%w and no system php found — run: pvm use <version>", err)
}
