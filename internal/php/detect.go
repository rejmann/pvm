// Package php asks php binaries about themselves: which ones exist outside
// pvm (DetectSystem) and what a given binary is (Probe).
package php

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/version"
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
	if runtime.GOOS == "windows" {
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

	slices.SortFunc(results, func(a, b SystemInstall) int {
		return version.Compare(a.Version, b.Version)
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
