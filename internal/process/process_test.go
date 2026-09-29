package process

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLookPath(t *testing.T) {
	shims := t.TempDir()
	sys := t.TempDir()
	shim := fakeExecutable(t, shims, "php")
	want := fakeExecutable(t, sys, "php")
	path := filepath.Dir(shim) + string(os.PathListSeparator) + filepath.Dir(want)

	if got := LookPath("php", path, shims); got != want {
		t.Errorf("LookPath skipping shims = %q, want %q", got, want)
	}
	if got := LookPath("php", path, ""); got != shim {
		t.Errorf("LookPath = %q, want %q", got, shim)
	}
	if got := LookPath("unzip", path, ""); got != "" {
		t.Errorf("LookPath(unzip) = %q, want none", got)
	}
}

func fakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, nil, 0755); err != nil {
		t.Fatal(err)
	}
	return p
}
