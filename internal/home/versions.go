package home

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rejmann/pvm/internal/version"
)

var ErrVersionNotInstalled = errors.New("version not installed")

func (d *Dir) versionsDir() string {
	return filepath.Join(d.Base, "versions")
}

func (d *Dir) VersionDir(v string) string {
	return filepath.Join(d.versionsDir(), v)
}

func (d *Dir) binaryFile(v string) string {
	return filepath.Join(d.VersionDir(v), "binary")
}

// RegisterVersion records bin as the php binary of version v, which makes v
// an installed version.
func (d *Dir) RegisterVersion(v, bin string) error {
	if err := os.MkdirAll(d.VersionDir(v), 0755); err != nil {
		return fmt.Errorf("create version directory: %w", err)
	}
	return os.WriteFile(d.binaryFile(v), []byte(bin), 0644)
}

func (d *Dir) RemoveVersionDir(v string) error {
	return os.RemoveAll(d.VersionDir(v))
}

// VersionBinary is the php binary registered for v.
func (d *Dir) VersionBinary(v string) (string, error) {
	data, err := os.ReadFile(d.binaryFile(v))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrVersionNotInstalled, v)
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// VersionInstalled reports whether v is registered and its binary exists.
func (d *Dir) VersionInstalled(v string) bool {
	bin, err := d.VersionBinary(v)
	if err != nil {
		return false
	}
	_, err = os.Stat(bin)
	return err == nil
}

// InstalledVersions lists the installed versions, oldest first.
func (d *Dir) InstalledVersions() ([]string, error) {
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
		if !d.VersionInstalled(e.Name()) {
			continue
		}
		versions = append(versions, e.Name())
	}

	slices.SortFunc(versions, version.Compare)
	return versions, nil
}

// MatchInstalled returns the installed version that satisfies v: an exact
// match first, otherwise — when v is a bare branch like "8.3" — the highest
// installed patch of that branch (e.g. "8.3.30").
func (d *Dir) MatchInstalled(v string) (string, bool) {
	if d.VersionInstalled(v) {
		return v, true
	}

	want, err := version.Parse(v)
	if err != nil || want.HasPatch() {
		return "", false
	}

	installed, err := d.InstalledVersions()
	if err != nil {
		return "", false
	}
	for i := len(installed) - 1; i >= 0; i-- {
		if version.Branch(installed[i]) == want.Branch() {
			return installed[i], true
		}
	}
	return "", false
}
