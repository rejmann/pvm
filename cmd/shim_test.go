package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rejmann/pvm/internal/symlink"
)

func TestShimTarget(t *testing.T) {
	t.Run("selected version wins over system php", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		setGlobal(t, m.Base, "8.3")
		sys := fakeExecutable(t, t.TempDir(), "php")

		got, err := shimTarget(m, t.TempDir(), "", filepath.Dir(sys))
		if err != nil {
			t.Fatal(err)
		}
		want, _ := m.GetVersionBinary("8.3")
		if got != want {
			t.Errorf("shimTarget = %q, want %q", got, want)
		}
	})

	t.Run("falls back to system php, skipping the shim dir", func(t *testing.T) {
		m := newManager(t)
		shim := fakeExecutable(t, symlink.ShimDir(m.Base), "php")
		sys := fakeExecutable(t, t.TempDir(), "php")
		path := filepath.Dir(shim) + string(os.PathListSeparator) + filepath.Dir(sys)

		got, err := shimTarget(m, t.TempDir(), "", path)
		if err != nil {
			t.Fatal(err)
		}
		if got != sys {
			t.Errorf("shimTarget = %q, want %q", got, sys)
		}
	})

	t.Run("no version and no system php", func(t *testing.T) {
		m := newManager(t)
		_, err := shimTarget(m, t.TempDir(), "", t.TempDir())
		if !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})

	t.Run("selected but not installed does not fall back", func(t *testing.T) {
		m := newManager(t)
		sys := fakeExecutable(t, t.TempDir(), "php")

		if _, err := shimTarget(m, t.TempDir(), "8.1", filepath.Dir(sys)); err == nil {
			t.Fatal("expected not-installed error instead of silently using system php")
		}
	})
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
