package php

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/rejmann/pvm/internal/version"
	"github.com/rejmann/pvm/system"
)

type SystemInstall struct {
	Version string
	Binary  string
}

func DetectSystem() []SystemInstall {
	globs := platformGlobs()
	seen := map[string]string{} // version → binary path

	for _, pattern := range globs {
		matches, _ := filepath.Glob(pattern)
		for _, bin := range matches {
			v := queryVersion(bin)
			if v == "" {
				continue
			}
			if _, exists := seen[v]; !exists {
				seen[v] = bin
			}
		}
	}

	phpBin := "php"
	if runtime.GOOS == system.Windows {
		phpBin = "php.exe"
	}
	if plain, err := exec.LookPath(phpBin); err == nil {
		if v := queryVersion(plain); v != "" {
			if _, exists := seen[v]; !exists {
				seen[v] = plain
			}
		}
	}

	var results []SystemInstall
	for v, bin := range seen {
		results = append(results, SystemInstall{Version: v, Binary: bin})
	}

	sort.Slice(results, func(i, j int) bool {
		a, errA := version.Parse(results[i].Version)
		b, errB := version.Parse(results[j].Version)
		if errA != nil || errB != nil {
			return results[i].Version < results[j].Version
		}
		return a.Compare(b) < 0
	})
	return results
}

func platformGlobs() []string {
	switch runtime.GOOS {
	case system.Windows:
		return windowsGlobs()
	case system.Darwin:
		return darwinGlobs()
	default:
		return linuxGlobs()
	}
}

func linuxGlobs() []string {
	return []string{
		"/usr/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
	}
}

func darwinGlobs() []string {
	return []string{
		"/opt/homebrew/opt/php*/bin/php", // Apple Silicon Homebrew
		"/usr/local/opt/php*/bin/php",    // Intel Homebrew
		"/opt/homebrew/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
		"/opt/local/bin/php[0-9]*", // MacPorts
	}
}

func windowsGlobs() []string {
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

func queryVersion(bin string) string {
	out, err := exec.Command(bin, "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;").Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if _, err := version.Parse(v); err != nil {
		return ""
	}
	return v
}
