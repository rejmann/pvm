//go:build windows

package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSwapMovesRunningBinaryAside(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pvm.exe")
	newBin := filepath.Join(dir, "new")
	for path, data := range map[string]string{exe: "old", newBin: "new", exe + ".old": "older"} {
		if err := os.WriteFile(path, []byte(data), 0755); err != nil {
			t.Fatal(err)
		}
	}

	if err := swap(newBin, exe); err != nil {
		t.Fatal(err)
	}

	if got, _ := os.ReadFile(exe); string(got) != "new" {
		t.Errorf("binary = %q, want new", got)
	}
	if got, _ := os.ReadFile(exe + ".old"); string(got) != "old" {
		t.Errorf("%s.old = %q, want the previous binary", exe, got)
	}
}

func TestRemoveOld(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pvm.exe")
	for _, name := range []string{"pvm.exe", "pvm.exe.old", "pvm.exe.old-123", "other.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0755); err != nil {
			t.Fatal(err)
		}
	}

	RemoveOld(exe)

	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 || names[0] != "other.exe" || names[1] != "pvm.exe" {
		t.Errorf("dir = %v, want [other.exe pvm.exe]", names)
	}
}
