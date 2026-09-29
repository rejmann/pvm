// Package sysphp finds PHP binaries installed on the system outside pvm.
package sysphp

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rejmann/pvm/internal/version"
)

// PHP is a php binary found on the system, outside pvm.
type PHP struct {
	Version string
	Binary  string
}

// Detect finds the PHP versions installed on the system, one binary per
// version, oldest first.
func Detect() []PHP {
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

	if plain, err := exec.LookPath(phpExe); err == nil {
		if v := queryVersion(plain); v != "" {
			if _, exists := seen[v]; !exists {
				seen[v] = plain
			}
		}
	}

	var results []PHP
	for v, bin := range seen {
		results = append(results, PHP{Version: v, Binary: bin})
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
