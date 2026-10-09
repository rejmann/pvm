// Package sysphp finds PHP binaries installed on the system outside pvm.
package sysphp

import (
	"context"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rejmann/pvm/internal/process"
	"github.com/rejmann/pvm/internal/version"
)

// queryTimeout bounds each php run, so a binary that hangs cannot stall pvm.
const queryTimeout = 10 * time.Second

// PHP is a php binary found on the system, outside pvm.
type PHP struct {
	Version string
	Binary  string
}

// Detect finds the PHP versions installed on the system, one binary per
// version, oldest first.
func Detect() []PHP {
	var bins []string
	for _, pattern := range platformGlobs() {
		matches, _ := filepath.Glob(pattern)
		bins = append(bins, matches...)
	}
	if plain, err := exec.LookPath(phpExe); err == nil {
		bins = append(bins, plain)
	}
	return detect(bins, queryVersion)
}

// detect asks every binary of bins for its version with query, all at once:
// each one is a php run, and they are independent. When several binaries
// report the same version, the first of bins is kept.
func detect(bins []string, query func(bin string) string) []PHP {
	versions := make([]string, len(bins))
	var wg sync.WaitGroup
	for i, bin := range bins {
		wg.Go(func() { versions[i] = query(bin) })
	}
	wg.Wait()

	seen := map[string]bool{}
	var results []PHP
	for i, v := range versions {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		results = append(results, PHP{Version: v, Binary: bins[i]})
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
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;")
	// Do not wait on output pipes a killed php may have left to a child.
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if _, err := version.Parse(v); err != nil {
		return ""
	}
	return v
}

// OnPath finds the php that runs from path when pvm selects nothing: the
// first one outside skip (pvm's shim directory), with symlinks resolved so
// it keeps pointing at the same version when the system default changes.
func OnPath(path, skip string) (PHP, bool) {
	bin := process.LookPath("php", path, skip)
	if bin == "" {
		return PHP{}, false
	}
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	v := queryVersion(bin)
	if v == "" {
		return PHP{}, false
	}
	return PHP{Version: v, Binary: bin}, true
}
