package composer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCanUnzip(t *testing.T) {
	if CanUnzip(t.TempDir()) {
		t.Error("found an archive tool in an empty PATH")
	}
	for _, name := range []string{"unzip", "7z"} {
		sys := fakeExecutable(t, t.TempDir(), name)
		if !CanUnzip(filepath.Dir(sys)) {
			t.Errorf("%s on PATH not found", name)
		}
	}
}

func TestForceANSI(t *testing.T) {
	tests := []struct {
		args     []string
		terminal bool
		noColor  string
		want     bool
	}{
		{args: []string{"install"}, terminal: true, want: true},
		{args: []string{"install"}, terminal: false},
		{args: []string{"install"}, terminal: true, noColor: "1"},
		{args: []string{"--no-ansi", "install"}, terminal: true},
		{args: []string{"install", "--ansi"}, terminal: true},
		{args: []string{"run", "x", "--", "--no-ansi"}, terminal: true, want: true},
	}
	for _, tt := range tests {
		if got := ForceANSI(tt.args, tt.terminal, tt.noColor); got != tt.want {
			t.Errorf("ForceANSI(%v, %v, %q) = %v, want %v", tt.args, tt.terminal, tt.noColor, got, tt.want)
		}
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
