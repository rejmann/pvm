package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if err := h.EnsureBaseDir(); err != nil {
		t.Fatal(err)
	}
	return h
}

// fakeBinary creates an empty executable standing in for php v.
func fakeBinary(t *testing.T, v string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// fakeInstall registers version v in h the same way pvm install does.
func fakeInstall(t *testing.T, h *home.Dir, v string) {
	t.Helper()
	if err := h.RegisterVersion(v, fakeBinary(t, v)); err != nil {
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

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

func TestVersionLabel(t *testing.T) {
	if got := versionLabel("8.4", true); got != "8.4 (lts)" {
		t.Errorf("alias label = %q", got)
	}
	if got := versionLabel("8.4", false); got != "8.4" {
		t.Errorf("plain label = %q", got)
	}
}

func TestConfirm(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, " YES \r\n": true, "n\n": false, "\n": false, "": false} {
		if got := confirm(strings.NewReader(input), &bytes.Buffer{}, "? "); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestRootCmd(t *testing.T) {
	root := newRootCmd("v1.2.3")
	if root.Version != "v1.2.3" {
		t.Errorf("Version = %q, want v1.2.3", root.Version)
	}

	want := []string{"available", "composer", "current", "install", "list", "remove",
		"run", "self-remove", "self-upgrade", "shim", "use", "which"}
	var got []string
	for _, c := range root.Commands() {
		got = append(got, c.Name())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("commands = %v, want %v", got, want)
	}

	for _, c := range root.Commands() {
		if (c.Name() == "shim") != c.Hidden {
			t.Errorf("%s: Hidden = %v", c.Name(), c.Hidden)
		}
	}
}
