package sysphp

import (
	"os"
	"path/filepath"
)

const phpExe = "php.exe"

func platformGlobs() []string {
	globs := []string{
		`C:\tools\php*\php.exe`,
		`C:\php*\php.exe`,
		`C:\ProgramData\chocolatey\lib\php*\tools\php.exe`,
	}

	if phprc := os.Getenv("PHPRC"); phprc != "" {
		globs = append(globs, filepath.Join(phprc, "php.exe"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		globs = append(globs,
			filepath.Join(home, "scoop", "apps", "php*", "current", "php.exe"),
		)
	}

	return globs
}
