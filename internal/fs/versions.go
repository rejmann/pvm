package fs

import (
	"errors"
	"io/fs"
	"os"
	"sort"

	"github.com/rejmann/pvm/internal/version"
)

func (m *Manager) InstalledVersions() ([]string, error) {
	entries, err := os.ReadDir(m.versionsDir())
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
		if !m.VersionInstalled(e.Name()) {
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

// MatchInstalled returns the installed version that satisfies v: an exact
// match first, otherwise — when v is a bare branch like "8.3" — the highest
// installed patch of that branch (e.g. "8.3.30").
func (m *Manager) MatchInstalled(v string) (string, bool) {
	if m.VersionInstalled(v) {
		return v, true
	}

	want, err := version.Parse(v)
	if err != nil || want.HasPatch() {
		return "", false
	}

	installed, err := m.InstalledVersions()
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
