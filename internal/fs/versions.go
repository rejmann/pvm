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
