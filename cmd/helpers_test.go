package cmd

import (
	"errors"
	"os"
	"path/filepath"
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
