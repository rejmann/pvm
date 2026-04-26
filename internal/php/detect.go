package php

import (
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
