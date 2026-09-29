package cmd

import (
	"errors"
	"os"

	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/pvm"
	"github.com/spf13/cobra"
)

// ShimCmd is invoked by the php shim script; it is not meant to be run by hand.
var ShimCmd = &cobra.Command{
	Use:                "shim php [args...]",
	Short:              "Run php with the version selected for the current directory",
	Hidden:             true,
	DisableFlagParsing: true,
	RunE:               runShim,
}

func runShim(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] != "php" {
		return errors.New("usage: pvm shim php [args...]")
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	bin, err := newManager(cmd).Shim(dir, os.Getenv(pvm.EnvVersion), os.Getenv("PATH"))
	if err != nil {
		return err
	}
	return process.Exec(bin, args[1:])
}
