//go:build windows

package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoveBinaryDeletesFileAndEmptyDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pvm dir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "pvm.exe")
	if err := os.WriteFile(exe, []byte("pvm"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := RemoveBinary(exe); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(exe); !os.IsNotExist(err) {
		t.Fatalf("%s should be moved aside right away, stat err = %v", exe, err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			entries, _ := os.ReadDir(dir)
			t.Fatalf("%s still exists with %d entries", dir, len(entries))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestRemoveBinaryMissingIsNoop(t *testing.T) {
	if err := RemoveBinary(filepath.Join(t.TempDir(), "pvm.exe")); err != nil {
		t.Fatal(err)
	}
}
