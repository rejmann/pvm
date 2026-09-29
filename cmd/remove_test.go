package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/composer"

	"github.com/rejmann/pvm/internal/home"
)

func TestRemoveVersion(t *testing.T) {
	t.Run("removes inactive version and its metadata", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		var gotVer string
		remove := func(_ *home.Dir, ver string) error { gotVer = ver; return nil }
		var out, errOut bytes.Buffer

		if err := removeVersion("8.2", h, remove, &out, &errOut); err != nil {
			t.Fatal(err)
		}
		if gotVer != "8.2" {
			t.Errorf("remover got %q, want 8.2", gotVer)
		}
		if _, err := os.Stat(h.VersionDir("8.2")); !os.IsNotExist(err) {
			t.Errorf("version dir should be gone, stat err = %v", err)
		}
		if !strings.Contains(out.String(), "PHP 8.2 removed.") {
			t.Errorf("unexpected output: %q", out.String())
		}
		if errOut.Len() != 0 {
			t.Errorf("no warning expected for inactive version, got %q", errOut.String())
		}
	})

	t.Run("removes only that version's Composer", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		mkdirAll(t, composer.VersionDir(h.ComposerDir(), "8.2"))
		mkdirAll(t, composer.VersionDir(h.ComposerDir(), "8.3"))
		remove := func(_ *home.Dir, _ string) error { return nil }

		if err := removeVersion("8.2", h, remove, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(composer.VersionDir(h.ComposerDir(), "8.2")); !os.IsNotExist(err) {
			t.Errorf("Composer of 8.2 should be gone, stat err = %v", err)
		}
		if _, err := os.Stat(composer.VersionDir(h.ComposerDir(), "8.3")); err != nil {
			t.Errorf("Composer of 8.3 must stay: %v", err)
		}
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		h := newHome(t)
		remove := func(_ *home.Dir, _ string) error { t.Error("remover must not run"); return nil }

		err := removeVersion("lts", h, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Fatalf("error = %v, want invalid version", err)
		}
	})

	t.Run("rejects version not installed", func(t *testing.T) {
		h := newHome(t)
		remove := func(_ *home.Dir, _ string) error { t.Error("remover must not run"); return nil }

		err := removeVersion("8.1", h, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("error = %v, want not installed", err)
		}
	})

	t.Run("keeps metadata when remover fails", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		boom := errors.New("permission denied")
		remove := func(_ *home.Dir, _ string) error { return boom }

		err := removeVersion("8.2", h, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
		if !h.Installed("8.2") {
			t.Error("version metadata should survive a failed removal")
		}
	})
}
