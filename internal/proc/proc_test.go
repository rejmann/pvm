package proc

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fakeExecutable(t *testing.T, dir, name string, mode os.FileMode) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, nil, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLookPathExcluding(t *testing.T) {
	skipped := fakeExecutable(t, t.TempDir(), "php", 0755)
	found := fakeExecutable(t, t.TempDir(), "php", 0755)
	path := filepath.Dir(skipped) + string(os.PathListSeparator) + filepath.Dir(found)

	if got := LookPathExcluding("php", path, filepath.Dir(skipped)); got != found {
		t.Errorf("LookPathExcluding = %q, want %q", got, found)
	}
	if got := LookPathExcluding("php", path, ""); got != skipped {
		t.Errorf("without skip = %q, want %q", got, skipped)
	}
	if got := LookPathExcluding("php", t.TempDir(), ""); got != "" {
		t.Errorf("empty dir = %q, want none", got)
	}

	if runtime.GOOS != "windows" {
		notExec := fakeExecutable(t, t.TempDir(), "php", 0644)
		if got := LookPathExcluding("php", filepath.Dir(notExec), ""); got != "" {
			t.Errorf("non-executable file found: %q", got)
		}
	}
}
