package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintCurrent(t *testing.T) {
	t.Run("no active version", func(t *testing.T) {
		m := newManager(t)
		var out bytes.Buffer

		if err := printCurrent(m, t.TempDir(), "", &out); err != nil {
			t.Fatal(err)
		}
		if got, want := out.String(), "No PHP version is currently active.\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("global version", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		setGlobal(t, m.Base, "8.3")
		var out bytes.Buffer

		if err := printCurrent(m, t.TempDir(), "", &out); err != nil {
			t.Fatal(err)
		}
		if got, want := out.String(), "Current PHP version: 8.3\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("project version shows its source", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.2")
		dir := t.TempDir()
		writePHPVersion(t, dir, "8.2")
		var out bytes.Buffer

		if err := printCurrent(m, dir, "", &out); err != nil {
			t.Fatal(err)
		}
		want := "Current PHP version: 8.2 (set by " + filepath.Join(dir, ".php-version") + ")\n"
		if out.String() != want {
			t.Errorf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("selected version not installed", func(t *testing.T) {
		m := newManager(t)
		setGlobal(t, m.Base, "8.1")

		err := printCurrent(m, t.TempDir(), "", &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("error = %v, want not installed", err)
		}
	})
}

func setGlobal(t *testing.T, base, v string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(v+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func writePHPVersion(t *testing.T, dir, v string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".php-version"), []byte(v+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
