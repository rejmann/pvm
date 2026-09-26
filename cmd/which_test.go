package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintWhich(t *testing.T) {
	m := newManager(t)
	fakeInstall(t, m, "8.3")
	dir := t.TempDir()
	writePHPVersion(t, dir, "8.3")
	var out bytes.Buffer

	if err := printWhich(m, dir, "", &out); err != nil {
		t.Fatal(err)
	}
	bin, _ := m.GetVersionBinary("8.3")
	if out.String() != bin+"\n" {
		t.Errorf("output = %q, want %q", out.String(), bin+"\n")
	}

	if err := printWhich(newManager(t), t.TempDir(), "", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "pvm use") {
		t.Errorf("nothing selected: error = %v", err)
	}
}
