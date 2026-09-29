package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/pvm"
)

// ltsResolver resolves the "lts" alias to itself.
type ltsResolver string

func (r ltsResolver) ResolveLTS() (string, error) { return string(r), nil }

// fakeSystem installs, removes and activates by only updating the pvm home.
type fakeSystem struct{ t *testing.T }

func (f fakeSystem) Install(h *home.Dir, ver string) error { fakeInstall(f.t, h, ver); return nil }
func (f fakeSystem) Remove(*home.Dir, string) error        { return nil }
func (f fakeSystem) Activate(h *home.Dir, ver, _ string) error {
	return h.SetCurrent(ver)
}
func (f fakeSystem) Deactivate(h *home.Dir) error { return h.ClearCurrent() }

// testManager runs the use cases on h without touching the system.
func testManager(t *testing.T, h *home.Dir) *pvm.Manager {
	t.Helper()
	return &pvm.Manager{Home: h, Installer: fakeSystem{t}, Activator: fakeSystem{t}, LTS: ltsResolver("8.4")}
}

func newHome(t *testing.T) *home.Dir {
	t.Helper()
	h := home.New(t.TempDir())
	if err := h.Init(); err != nil {
		t.Fatal(err)
	}
	return h
}

// fakeInstall registers version v in h the same way the real installers do:
// a recorded binary pointing at an existing executable.
func fakeInstall(t *testing.T, h *home.Dir, v string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := h.SetBinary(v, bin); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}
