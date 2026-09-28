//go:build linux || darwin

package proc

import (
	"fmt"
	"os"
	"syscall"
)

// Exec replaces the current process with bin, so signals, stdio and the
// exit code behave exactly as if bin had been run directly.
func Exec(bin string, args []string) error {
	argv := append([]string{bin}, args...)
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		return fmt.Errorf("exec %s: %w", bin, err)
	}
	return nil
}
