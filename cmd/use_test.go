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
