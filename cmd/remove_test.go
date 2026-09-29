package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveVersionOutput(t *testing.T) {
	t.Run("inactive version", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		var out, errOut bytes.Buffer

		if err := removeVersion(testManager(t, h), "8.2", &out, &errOut); err != nil {
			t.Fatal(err)
		}
		if out.String() != "PHP 8.2 removed.\n" || errOut.Len() != 0 {
			t.Errorf("out = %q, errOut = %q", out.String(), errOut.String())
		}
	})

	t.Run("version installed outside pvm stays on the system", func(t *testing.T) {
		h := newHome(t)
		bin := filepath.Join(t.TempDir(), "php8.3")
		if err := os.WriteFile(bin, nil, 0755); err != nil {
			t.Fatal(err)
		}
		if err := h.SetSystem("8.3", bin); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer

		if err := removeVersion(testManager(t, h), "8.3", &out, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if want := "PHP 8.3 is no longer managed by pvm; it was installed outside pvm and stays on the system.\n"; out.String() != want {
			t.Errorf("out = %q, want %q", out.String(), want)
		}
	})

	t.Run("warns when it was the active version", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.2")
		setGlobal(t, h, "8.2")
		var out, errOut bytes.Buffer

		if err := removeVersion(testManager(t, h), "8.2", &out, &errOut); err != nil {
			t.Fatal(err)
		}
		if want := "Warning: PHP 8.2 was the active version. No version is now active.\n"; errOut.String() != want {
			t.Errorf("errOut = %q, want %q", errOut.String(), want)
		}
	})
}
