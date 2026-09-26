//go:build linux || darwin

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

// execBinary replaces the current process with bin, so signals, stdio and the
// exit code behave exactly as if bin had been run directly.
func execBinary(bin string, args []string) error {
	argv := append([]string{bin}, args...)
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		return fmt.Errorf("exec %s: %w", bin, err)
	}
	return nil
}
