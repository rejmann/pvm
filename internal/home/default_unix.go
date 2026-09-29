//go:build !windows

package home

import (
	"os"
	"path/filepath"
)

func defaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pvm")
}
