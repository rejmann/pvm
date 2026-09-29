// Package composer holds the helpers of pvm composer.
package composer

import (
	"os"

	"github.com/rejmann/pvm/internal/process"
	"golang.org/x/term"
)

// CanUnzip reports whether Composer can extract zips without the PHP
// extension: it also accepts unzip or 7-Zip on PATH.
func CanUnzip(path string) bool {
	for _, name := range []string{"unzip", "7z", "7zz"} {
		if process.LookPath(name, path, "") != "" {
			return true
		}
	}
	return false
}

// ForceANSI reports whether to pass --ansi: Composer's stderr goes through
// pvm (to spot missing extensions), so Composer would otherwise turn colors
// off even in a terminal.
func ForceANSI(args []string, stdoutTerminal bool, noColor string) bool {
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

// IsTerminal is false for pipes and redirects, including </dev/null (a char
// device, so a mode check alone is not enough).
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
