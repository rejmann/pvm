//go:build windows

package home

import (
	"os"
	"path/filepath"
)

func defaultPath() string {
	// %LOCALAPPDATA% → C:\Users\<user>\AppData\Local
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "pvm")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "AppData", "Local", "pvm")
}
