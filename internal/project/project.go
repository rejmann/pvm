// Package project reads and writes the per-project .php-version file.
package project

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = ".php-version"

var ErrNotFound = errors.New(FileName + " not found")

// Find walks up from dir to the filesystem root and returns the version
// declared in the nearest .php-version file, along with that file's path.
func Find(dir string) (version, path string, err error) {
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	for {
		p := filepath.Join(dir, FileName)
		v, err := Read(p)
		if err == nil {
			return v, p, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", ErrNotFound
		}
		dir = parent
	}
}

// Read returns the first non-empty, non-comment line of a .php-version file.
func Read(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, nil
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return "", fmt.Errorf("%s is empty", path)
}

// Write creates or replaces the .php-version file in dir.
func Write(dir, version string) (string, error) {
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(version+"\n"), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", p, err)
	}
	return p, nil
}

// Remove deletes the .php-version file in dir. It is a no-op if none exists.
func Remove(dir string) (string, error) {
	p := filepath.Join(dir, FileName)
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove %s: %w", p, err)
	}
	return p, nil
}
