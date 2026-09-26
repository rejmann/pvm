package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	phpfs "github.com/rejmann/pvm/internal/fs"
)

type fakeResolver struct {
	v   string
	err error
}

func (f fakeResolver) ResolveLTS() (string, error) { return f.v, f.err }

// failResolver fails the test if the lts alias is resolved when it shouldn't be.
type failResolver struct{ t *testing.T }

func (f failResolver) ResolveLTS() (string, error) {
	f.t.Helper()
	f.t.Error("ResolveLTS called unexpectedly")
	return "", errors.New("unexpected")
}

func newManager(t *testing.T) *phpfs.Manager {
	t.Helper()
	m := phpfs.NewManager(t.TempDir())
	if err := m.EnsurebaseDir(); err != nil {
		t.Fatal(err)
	}
	return m
}

// fakeInstall registers version v in m the same way the real installers do:
// a versions/<v>/binary file pointing at an existing executable.
func fakeInstall(t *testing.T, m *phpfs.Manager, v string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(m.VersionDir(v), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.VersionDir(v), "binary"), []byte(bin), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}
