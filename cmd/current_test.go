package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPrintCurrent(t *testing.T) {
	t.Run("no active version", func(t *testing.T) {
		m := newManager(t)
		var out bytes.Buffer

		if err := printCurrent(m, &out); err != nil {
			t.Fatal(err)
		}
		if got, want := out.String(), "No PHP version is currently active.\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("active version", func(t *testing.T) {
		m := newManager(t)
		if err := os.WriteFile(filepath.Join(m.Base, "current-version"), []byte("8.3\n"), 0644); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer

		if err := printCurrent(m, &out); err != nil {
			t.Fatal(err)
		}
		if got, want := out.String(), "Current PHP version: 8.3\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})
}
