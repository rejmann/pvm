// Package proc runs other programs: package managers and PowerShell whose
// output the user sees, and php itself, which replaces pvm (Exec).
package proc

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Streams is where the output of the programs pvm runs goes.
type Streams struct {
	Out, Err io.Writer
}

// Std is the terminal pvm itself writes to.
func Std() Streams {
	return Streams{Out: os.Stdout, Err: os.Stderr}
}

// Run runs name with its output on s and stdin inherited, so sudo and
// package managers can prompt.
func (s Streams) Run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, s.Out, s.Err
	return c.Run()
}

// Quiet runs name discarding its output; only success matters.
func Quiet(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// LookPathExcluding is exec.LookPath over path, skipping the entry skip
// (e.g. pvm's shim directory, so the shim never finds itself).
func LookPathExcluding(name, path, skip string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	skip = filepath.Clean(skip)

	for _, dir := range filepath.SplitList(path) {
		if dir == "" || strings.EqualFold(filepath.Clean(dir), skip) {
			continue
		}
		p := filepath.Join(dir, name)
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && fi.Mode()&0111 == 0 {
			continue
		}
		return p
	}
	return ""
}
