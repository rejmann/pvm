package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	root := NewRootCmd("v1.2.3")

	var names []string
	for _, c := range root.Commands() {
		names = append(names, c.Name())
	}
	for _, want := range []string{"available", "install", "list", "use", "remove", "current", "which", "run", "composer", "shim", "self-upgrade", "self-remove"} {
		if !strings.Contains(" "+strings.Join(names, " ")+" ", " "+want+" ") {
			t.Errorf("missing command %q in %v", want, names)
		}
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "v1.2.3") {
		t.Errorf("--version output = %q", out.String())
	}
}
