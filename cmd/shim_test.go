package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rejmann/pvm/internal/pvm"
)

func TestShimTarget(t *testing.T) {
	t.Run("selected version wins over system php", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		setGlobal(t, h, "8.3")
		sys := fakeExecutable(t, t.TempDir(), "php")

		got, err := shimTarget(testManager(t, h), t.TempDir(), "", filepath.Dir(sys))
		if err != nil {
			t.Fatal(err)
		}
		want, _ := h.Binary("8.3")
		if got != want {
			t.Errorf("shimTarget = %q, want %q", got, want)
		}
	})

	t.Run("falls back to system php, skipping the shim dir", func(t *testing.T) {
		h := newHome(t)
		shim := fakeExecutable(t, h.ShimDir(), "php")
		sys := fakeExecutable(t, t.TempDir(), "php")
		path := filepath.Dir(shim) + string(os.PathListSeparator) + filepath.Dir(sys)

		got, err := shimTarget(testManager(t, h), t.TempDir(), "", path)
		if err != nil {
			t.Fatal(err)
		}
		if got != sys {
			t.Errorf("shimTarget = %q, want %q", got, sys)
		}
	})

	t.Run("no version and no system php", func(t *testing.T) {
		h := newHome(t)
		_, err := shimTarget(testManager(t, h), t.TempDir(), "", t.TempDir())
		if !errors.Is(err, pvm.ErrNoActiveVersion) {
			t.Fatalf("error = %v, want pvm.ErrNoActiveVersion", err)
		}
	})

	t.Run("selected but not installed does not fall back", func(t *testing.T) {
		h := newHome(t)
		sys := fakeExecutable(t, t.TempDir(), "php")

		if _, err := shimTarget(testManager(t, h), t.TempDir(), "8.1", filepath.Dir(sys)); err == nil {
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
