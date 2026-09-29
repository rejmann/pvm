// Package process runs the programs pvm hands over to (php, Composer) and
// finds programs on PATH.
package process

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// Run runs bin as a child with pvm's stdin, stdout and environment plus env,
// and its stderr copied both to pvm's stderr and to watch. It returns the
// child's exit code. Ctrl+C reaches the child directly (same process group),
// so pvm only ignores it; other termination signals are forwarded.
func Run(bin string, args []string, env map[string]string, watch io.Writer) (int, error) {
	c := exec.Command(bin, args...)
	c.Stdin, c.Stdout = os.Stdin, os.Stdout
	c.Stderr = io.MultiWriter(os.Stderr, watch)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(sig)
		close(sig)
	}()

	if err := c.Start(); err != nil {
		return 0, fmt.Errorf("run %s: %w", bin, err)
	}
	go func() {
		for s := range sig {
			if s != os.Interrupt {
				_ = c.Process.Signal(s)
			}
		}
	}()

	err := c.Wait()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		if code := exitErr.ExitCode(); code >= 0 {
			return code, nil
		}
		return 1, nil // killed by a signal
	case err != nil:
		return 0, fmt.Errorf("run %s: %w", bin, err)
	}
	return 0, nil
}

// LookPath is exec.LookPath over the directory list path, skipping the
// directory skip (e.g. pvm's own shims). It returns "" when name is not found.
func LookPath(name, path, skip string) string {
	name += exeSuffix
	skip = filepath.Clean(skip)

	for _, dir := range filepath.SplitList(path) {
		if dir == "" || strings.EqualFold(filepath.Clean(dir), skip) {
			continue
		}
		p := filepath.Join(dir, name)
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() || !isExecutable(fi) {
			continue
		}
		return p
	}
	return ""
}
