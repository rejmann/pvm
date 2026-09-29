package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestUseVersionOutput(t *testing.T) {
	h := newHome(t)
	fakeInstall(t, h, "8.4")
	var out bytes.Buffer

	if err := useVersion(testManager(t, h), "lts", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "Now using PHP 8.4 (lts).\n") {
		t.Errorf("output = %q", out.String())
	}
	if v, _ := h.Current(); v != "8.4" {
		t.Errorf("current = %q, want 8.4", v)
	}
}

func TestUseArg(t *testing.T) {
	dir := t.TempDir()

	got, err := useArg([]string{"8.1"}, dir, &bytes.Buffer{})
	if err != nil || got != "8.1" {
		t.Fatalf("explicit arg: (%q, %v)", got, err)
	}

	if _, err := useArg(nil, dir, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "no .php-version found") {
		t.Fatalf("no file: error = %v", err)
	}

	writePHPVersion(t, dir, "8.2")
	var out bytes.Buffer
	got, err = useArg(nil, dir, &out)
	if err != nil || got != "8.2" {
		t.Fatalf("from file: (%q, %v)", got, err)
	}
	if !strings.Contains(out.String(), "with version 8.2") {
		t.Errorf("unexpected output: %q", out.String())
	}
}
