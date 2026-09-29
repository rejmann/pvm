package pvm

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rejmann/pvm/internal/home"
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

// fakeInstaller records calls; Install registers the version like the real
// installers do, unless err is set.
type fakeInstaller struct {
	t                  *testing.T
	installed, removed []string
	err                error
}

func (f *fakeInstaller) Install(h *home.Dir, ver string) error {
	f.installed = append(f.installed, ver)
	if f.err != nil {
		return f.err
	}
	fakeInstall(f.t, h, ver)
	return nil
}

func (f *fakeInstaller) Remove(_ *home.Dir, ver string) error {
	f.removed = append(f.removed, ver)
	return f.err
}

type fakeActivator struct {
	activated   []string // "version=binary"
	deactivated int
	err         error
}

func (f *fakeActivator) Activate(h *home.Dir, ver, bin string) error {
	if f.err != nil {
		return f.err
	}
	f.activated = append(f.activated, ver+"="+bin)
	return h.SetCurrent(ver)
}

func (f *fakeActivator) Deactivate(h *home.Dir) error {
	f.deactivated++
	return h.ClearCurrent()
}

// newTestManager returns a Manager on an empty pvm home, with fakes that
// fail the test if the lts alias is resolved.
func newTestManager(t *testing.T) (*Manager, *fakeInstaller, *fakeActivator) {
	t.Helper()
	h := home.New(t.TempDir())
	if err := h.Init(); err != nil {
		t.Fatal(err)
	}
	inst := &fakeInstaller{t: t}
	act := &fakeActivator{}
	return &Manager{Home: h, Installer: inst, Activator: act, LTS: failResolver{t}}, inst, act
}

// fakeInstall registers version v in h, pointing at an existing executable.
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

func setGlobal(t *testing.T, h *home.Dir, v string) {
	t.Helper()
	if err := h.SetCurrent(v); err != nil {
		t.Fatal(err)
	}
}

func writePHPVersion(t *testing.T, dir, v string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".php-version"), []byte(v+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

func fakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	mkdirAll(t, dir)
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, nil, 0755); err != nil {
		t.Fatal(err)
	}
	return p
}
