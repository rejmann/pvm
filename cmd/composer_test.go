package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasArchiveTool(t *testing.T) {
	if hasArchiveTool(t.TempDir()) {
		t.Error("found an archive tool in an empty PATH")
	}
	for _, name := range []string{"unzip", "7z"} {
		sys := fakeExecutable(t, t.TempDir(), name)
		if !hasArchiveTool(filepath.Dir(sys)) {
			t.Errorf("%s on PATH not found", name)
		}
	}
}

func TestOfferZipExtensionWithoutInstall(t *testing.T) {
	t.Run("not a terminal", func(t *testing.T) {
		var out bytes.Buffer
		offerZipExtension(newHome(t), "8.3", false, strings.NewReader("y\n"), &out)
		if !strings.Contains(out.String(), "PHP 8.3 has no zip extension") ||
			!strings.Contains(out.String(), "in a terminal") {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("declined", func(t *testing.T) {
		var out bytes.Buffer
		offerZipExtension(newHome(t), "8.3", true, strings.NewReader("n\n"), &out)
		if strings.Contains(out.String(), "installed") || strings.Contains(out.String(), "Warning") {
			t.Errorf("output = %q", out.String())
		}
	})
}
