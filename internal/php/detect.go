package php

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rejmann/pvm/internal/version"
)

// SystemInstall represents a PHP binary found on the host system.
type SystemInstall struct {
	Version string
	Binary  string
}

// DetectSystem scans well-known locations for PHP binaries installed outside
// of pvm and returns one entry per unique version found.
func DetectSystem() []SystemInstall {
	globs := []string{
		"/usr/bin/php[0-9]*",
		"/usr/local/bin/php[0-9]*",
		"/opt/homebrew/bin/php[0-9]*",
		"/opt/homebrew/opt/php*/bin/php",
	}

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

	// Also check the plain `php` in PATH in case it points somewhere not covered.
	if plain, err := exec.LookPath("php"); err == nil {
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

// queryVersion runs the given PHP binary and returns its minor version string
// (e.g. "8.4"), or empty string on failure.
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
