package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestInstallVersionOutput(t *testing.T) {
	t.Run("concrete version", func(t *testing.T) {
		var out bytes.Buffer
		if err := installVersion(testManager(t, newHome(t)), "8.3", &out); err != nil {
			t.Fatal(err)
		}
		if want := "Installing PHP 8.3...\nPHP 8.3 installed successfully.\n"; out.String() != want {
			t.Errorf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("labels the lts alias", func(t *testing.T) {
		var out bytes.Buffer
		if err := installVersion(testManager(t, newHome(t)), "lts", &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "PHP 8.4 (lts) installed successfully.") {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("prints nothing when already installed", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		var out bytes.Buffer
		if err := installVersion(testManager(t, h), "8.3", &out); err == nil {
			t.Fatal("expected an error")
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want none", out.String())
		}
	})
}
