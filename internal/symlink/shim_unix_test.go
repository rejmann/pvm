//go:build linux || darwin

package symlink

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimScriptRunsPvm(t *testing.T) {
	dir := t.TempDir()

	// A stand-in for pvm, at a path with a space and a quote, that echoes its args.
	fakePvm := filepath.Join(dir, "my 'pvm'")
	if err := os.WriteFile(fakePvm, []byte("#!/bin/sh\nprintf '%s|' \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}

	shim := filepath.Join(dir, "bin", "php")
	if err := writeShim(shim, fakePvm); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(shim)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&0111 == 0 {
		t.Errorf("shim is not executable: %v", fi.Mode())
	}

	out, err := exec.Command(shim, "-r", "echo 'hi there';").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), "shim|php|-r|echo 'hi there';|"; got != want {
		t.Errorf("shim forwarded %q, want %q", got, want)
	}
}

func TestWriteShimReplacesSymlink(t *testing.T) {
	dir := t.TempDir()
	shim := filepath.Join(dir, "php")
	// Older pvm versions left a plain symlink to the php binary here.
	if err := os.Symlink("/usr/bin/php8.3", shim); err != nil {
		t.Fatal(err)
	}

	if err := writeShim(shim, "/usr/local/bin/pvm"); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Lstat(shim)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("shim is still a symlink")
	}
	data, _ := os.ReadFile(shim)
	if !strings.Contains(string(data), "exec '/usr/local/bin/pvm' shim php \"$@\"") {
		t.Errorf("unexpected shim content:\n%s", data)
	}
}
