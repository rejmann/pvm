package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRemoveVersion(t *testing.T) {
	t.Run("removes inactive version and its metadata", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.2")
		var gotVer string
		remove := func(_, ver string) error { gotVer = ver; return nil }
		var out, errOut bytes.Buffer

		if err := removeVersion("8.2", m, remove, &out, &errOut); err != nil {
			t.Fatal(err)
		}
		if gotVer != "8.2" {
			t.Errorf("remover got %q, want 8.2", gotVer)
		}
		if _, err := os.Stat(m.VersionDir("8.2")); !os.IsNotExist(err) {
			t.Errorf("version dir should be gone, stat err = %v", err)
		}
		if !strings.Contains(out.String(), "PHP 8.2 removed.") {
			t.Errorf("unexpected output: %q", out.String())
		}
		if errOut.Len() != 0 {
			t.Errorf("no warning expected for inactive version, got %q", errOut.String())
		}
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		m := newManager(t)
		remove := func(_, _ string) error { t.Error("remover must not run"); return nil }

		err := removeVersion("lts", m, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Fatalf("error = %v, want invalid version", err)
		}
	})

	t.Run("rejects version not installed", func(t *testing.T) {
		m := newManager(t)
		remove := func(_, _ string) error { t.Error("remover must not run"); return nil }

		err := removeVersion("8.1", m, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("error = %v, want not installed", err)
		}
	})

	t.Run("keeps metadata when remover fails", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.2")
		boom := errors.New("permission denied")
		remove := func(_, _ string) error { return boom }

		err := removeVersion("8.2", m, remove, &bytes.Buffer{}, &bytes.Buffer{})
		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
		if !m.VersionInstalled("8.2") {
			t.Error("version metadata should survive a failed removal")
		}
	})
}
