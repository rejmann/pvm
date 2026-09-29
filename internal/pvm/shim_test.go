package pvm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestShim(t *testing.T) {
	t.Run("selected version wins over system php", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		h := m.Home
		fakeInstall(t, h, "8.3")
		setGlobal(t, h, "8.3")
		sys := fakeExecutable(t, t.TempDir(), "php")

		got, err := m.Shim(t.TempDir(), "", filepath.Dir(sys))
		if err != nil {
			t.Fatal(err)
		}
		want, _ := h.Binary("8.3")
		if got != want {
			t.Errorf("Shim = %q, want %q", got, want)
		}
	})

	t.Run("falls back to system php, skipping the shim dir", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		h := m.Home
		shim := fakeExecutable(t, h.ShimDir(), "php")
		sys := fakeExecutable(t, t.TempDir(), "php")
		path := filepath.Dir(shim) + string(os.PathListSeparator) + filepath.Dir(sys)

		got, err := m.Shim(t.TempDir(), "", path)
		if err != nil {
			t.Fatal(err)
		}
		if got != sys {
			t.Errorf("Shim = %q, want %q", got, sys)
		}
	})

	t.Run("no version and no system php", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		_, err := m.Shim(t.TempDir(), "", t.TempDir())
		if !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})

	t.Run("selected but not installed does not fall back", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		sys := fakeExecutable(t, t.TempDir(), "php")

		if _, err := m.Shim(t.TempDir(), "8.1", filepath.Dir(sys)); err == nil {
			t.Fatal("expected not-installed error instead of silently using system php")
		}
	})
}
