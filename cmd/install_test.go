package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestInstallVersion(t *testing.T) {
	t.Run("installs concrete version", func(t *testing.T) {
		m := newManager(t)
		var gotBase, gotVer string
		install := func(base, ver string) error {
			gotBase, gotVer = base, ver
			return nil
		}
		var out bytes.Buffer

		if err := installVersion("8.3", m, failResolver{t}, install, &out, &out); err != nil {
			t.Fatal(err)
		}
		if gotBase != m.Base || gotVer != "8.3" {
			t.Errorf("installer called with (%q, %q), want (%q, \"8.3\")", gotBase, gotVer, m.Base)
		}
		if !strings.Contains(out.String(), "PHP 8.3 installed successfully.") {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("resolves lts alias", func(t *testing.T) {
		m := newManager(t)
		var gotVer string
		install := func(_, ver string) error { gotVer = ver; return nil }
		var out bytes.Buffer

		if err := installVersion("lts", m, fakeResolver{v: "8.4"}, install, &out, &out); err != nil {
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
		m := newManager(t)
		install := func(_, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("8.x", m, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Fatalf("error = %v, want invalid version", err)
		}
	})

	t.Run("rejects already installed", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		install := func(_, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("8.3", m, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "already installed") {
			t.Fatalf("error = %v, want already installed", err)
		}
	})

	t.Run("propagates resolver error", func(t *testing.T) {
		m := newManager(t)
		boom := errors.New("offline")
		install := func(_, _ string) error { t.Error("installer must not run"); return nil }

		err := installVersion("lts", m, fakeResolver{err: boom}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})

	t.Run("propagates installer error", func(t *testing.T) {
		m := newManager(t)
		boom := errors.New("apt failed")
		install := func(_, _ string) error { return boom }

		err := installVersion("8.3", m, failResolver{t}, install, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}
