package use

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArg(t *testing.T) {
	dir := t.TempDir()

	got, err := Arg([]string{"8.1"}, dir, &bytes.Buffer{})
	if err != nil || got != "8.1" {
		t.Fatalf("explicit arg: (%q, %v)", got, err)
	}

	if _, err := Arg(nil, dir, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "no .php-version found") {
		t.Fatalf("no file: error = %v", err)
	}

	writePHPVersion(t, dir, "8.2")
	var out bytes.Buffer
	got, err = Arg(nil, dir, &out)
	if err != nil || got != "8.2" {
		t.Fatalf("from file: (%q, %v)", got, err)
	}
	if !strings.Contains(out.String(), "with version 8.2") {
		t.Errorf("unexpected output: %q", out.String())
	}
}

func writePHPVersion(t *testing.T, dir, v string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".php-version"), []byte(v+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
