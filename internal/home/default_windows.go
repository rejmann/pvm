//go:build windows

package home

import (
	"os"
	"path/filepath"
)

// defaultBase is %LOCALAPPDATA%\pvm (C:\Users\<user>\AppData\Local\pvm).
func defaultBase() string {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "pvm")
	}

	home, _ := os.UserHomeDir()

	return filepath.Join(home, "AppData", "Local", "pvm")
}
