//go:build linux || darwin

package home

import (
	"os"
	"path/filepath"
)

// defaultBase is ~/.pvm.
func defaultBase() string {
	home, _ := os.UserHomeDir()

	return filepath.Join(home, ".pvm")
}
