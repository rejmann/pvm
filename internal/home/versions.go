package home

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rejmann/pvm/internal/version"
)

var ErrVersionNotInstalled = errors.New("version not installed")

func (d *Dir) binaryFile(v string) string {
	return filepath.Join(d.VersionDir(v), "binary")
}

// Binary returns the php binary recorded for installed version v.
func (d *Dir) Binary(v string) (string, error) {
	data, err := os.ReadFile(d.binaryFile(v))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrVersionNotInstalled, v)
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetBinary records bin as the php binary of version v, marking v installed.
func (d *Dir) SetBinary(v, bin string) error {
	if err := os.MkdirAll(d.VersionDir(v), 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}
	return os.WriteFile(d.binaryFile(v), []byte(bin), 0644)
}

// Installed reports whether v is recorded and its binary still exists.
func (d *Dir) Installed(v string) bool {
	bin, err := d.Binary(v)
	if err != nil {
		return false
	}
	_, err = os.Stat(bin)
	return err == nil
}

// RemoveVersion deletes the metadata of version v.
func (d *Dir) RemoveVersion(v string) error {
	return os.RemoveAll(d.VersionDir(v))
}

// Versions lists the installed versions, oldest first.
func (d *Dir) Versions() ([]string, error) {
	entries, err := os.ReadDir(d.versionsDir())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := version.Parse(e.Name()); err != nil {
			continue
		}
		if !d.Installed(e.Name()) {
			continue
		}
		versions = append(versions, e.Name())
	}

	sort.Slice(versions, func(i, j int) bool {
		a, _ := version.Parse(versions[i])
		b, _ := version.Parse(versions[j])
		return a.Compare(b) < 0
	})
	return versions, nil
}

// Match returns the installed version that satisfies v: an exact match
// first, otherwise — when v is a bare branch like "8.3" — the highest
// installed patch of that branch (e.g. "8.3.30").
func (d *Dir) Match(v string) (string, bool) {
	if d.Installed(v) {
		return v, true
	}

	want, err := version.Parse(v)
	if err != nil || want.HasPatch() {
		return "", false
	}

	installed, err := d.Versions()
	if err != nil {
		return "", false
	}
	for i := len(installed) - 1; i >= 0; i-- {
		got, _ := version.Parse(installed[i])
		if got.Major == want.Major && got.Minor == want.Minor {
			return installed[i], true
		}
	}
	return "", false
}
