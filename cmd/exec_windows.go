//go:build windows

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// execBinary runs bin with inherited stdio and exits with its exit code.
func execBinary(bin string, args []string) error {
	c := exec.Command(bin, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := c.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		return fmt.Errorf("run %s: %w", bin, err)
	}
	os.Exit(0)
	return nil
}
