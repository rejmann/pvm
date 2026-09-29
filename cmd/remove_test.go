package cmd

import (
	"bytes"
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
