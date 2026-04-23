package cmd

import (
	"os"
	"path/filepath"
)

func baseDir() string {
	if v := os.Getenv("PVM_HOME"); v != "" {
		return v
	}

	home, _ := os.UserHomeDir()

	return filepath.Join(home, ".pvm")
}
