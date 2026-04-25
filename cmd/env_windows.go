//go:build windows

package cmd

import (
	"os"
	"path/filepath"
)

func baseDir() string {
	if v := os.Getenv("PVM_HOME"); v != "" {
		return v
	}

	// %LOCALAPPDATA% → C:\Users\<user>\AppData\Local
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "pvm")
	}

	home, _ := os.UserHomeDir()

	return filepath.Join(home, "AppData", "Local", "pvm")
}
