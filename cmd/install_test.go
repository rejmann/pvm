package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/home"
)

func TestInstallVersion(t *testing.T) {
	t.Run("installs concrete version", func(t *testing.T) {
		h := newHome(t)
		var gotHome *home.Dir
		var gotVer string
		install := func(h *home.Dir, ver string) error {
			gotHome, gotVer = h, ver
			return nil
		}
		var out bytes.Buffer

		if err := installVersion("8.3", h, failResolver{t}, install, &out, &out); err != nil {
			t.Fatal(err)
		}
		if gotHome != h || gotVer != "8.3" {
			t.Errorf("installer called with (%v, %q), want (%v, \"8.3\")", gotHome, gotVer, h)
		}
		if !strings.Contains(out.String(), "PHP 8.3 installed successfully.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("resolves lts alias", func(t *testing.T) {
		h := newHome(t)
		var gotVer string
		install := func(_ *home.Dir, ver string) error { gotVer = ver; return nil }
		var out bytes.Buffer

		if err := installVersion("lts", h, fakeResolver{v: "8.4"}, install, &out, &out); err != nil {
			t.Fatal(err)
		}
		if gotVer != "8.4" {
			t.Errorf("installer got %q, want 8.4", gotVer)
		}
		if !strings.Contains(out.String(), "8.4 (lts)") {
			t.Errorf("output should label the alias: %q", out.String())
		}
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		h := newHome(t)
		install := func(_ *home.Dir, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("8.x", h, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Fatalf("error = %v, want invalid version", err)
		}
	})

	t.Run("rejects already installed", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		install := func(_ *home.Dir, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("8.3", h, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "already installed") {
			t.Fatalf("error = %v, want already installed", err)
		}
	})

	t.Run("propagates resolver error", func(t *testing.T) {
		h := newHome(t)
		boom := errors.New("offline")
		install := func(_ *home.Dir, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("lts", h, fakeResolver{err: boom}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})

	t.Run("propagates installer error", func(t *testing.T) {
		h := newHome(t)
		boom := errors.New("apt failed")
		install := func(_ *home.Dir, _ string) error { return boom }

		err := installVersion("8.3", h, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}
