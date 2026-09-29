package run

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		in          []string
		wantVersion string
		wantRest    []string
		wantErr     string
	}{
		{in: nil, wantVersion: "", wantRest: nil},
		{in: []string{"8.5", "script.php"}, wantVersion: "8.5", wantRest: []string{"script.php"}},
		{in: []string{"8.2.30", "--", "-v"}, wantVersion: "8.2.30", wantRest: []string{"-v"}},
		{in: []string{"lts", "script.php"}, wantVersion: "lts", wantRest: []string{"script.php"}},
		{in: []string{"script.php", "8.5"}, wantVersion: "", wantRest: []string{"script.php", "8.5"}},
		{in: []string{"-r", "echo 1;"}, wantVersion: "", wantRest: []string{"-r", "echo 1;"}},
		{in: []string{"--", "8.5"}, wantVersion: "", wantRest: []string{"8.5"}},
		{in: []string{"8"}, wantVersion: "", wantRest: []string{"8"}},

		// -v / --version flags
		{in: []string{"-v", "8.2", "script.php"}, wantVersion: "8.2", wantRest: []string{"script.php"}},
		{in: []string{"--version", "lts", "script.php", "a"}, wantVersion: "lts", wantRest: []string{"script.php", "a"}},
		{in: []string{"--version=8.2", "script.php"}, wantVersion: "8.2", wantRest: []string{"script.php"}},
		{in: []string{"script.php", "-v", "8.2"}, wantVersion: "8.2", wantRest: []string{"script.php"}},
		{in: []string{"script.php", "a", "--version", "8.2", "b"}, wantVersion: "8.2", wantRest: []string{"script.php", "a", "b"}},
		{in: []string{"script.php", "--", "-v", "8.2"}, wantVersion: "", wantRest: []string{"script.php", "-v", "8.2"}},
		{in: []string{"script.php", "-v"}, wantErr: "flag -v needs a version"},
		{in: []string{"-v", "8.2", "--", "8.5"}, wantVersion: "8.2", wantRest: []string{"8.5"}},
		{in: []string{"-v"}, wantErr: "flag -v needs a version"},
		{in: []string{"-v", "8.2", "8.5", "script.php"}, wantErr: "version given twice"},
		{in: []string{"-v", "8.2", "--version", "8.5", "x.php"}, wantErr: "version given twice"},
	}

	for _, tt := range tests {
		v, rest, err := SplitArgs(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("SplitArgs(%q) error = %v, want %q", tt.in, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("SplitArgs(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if v != tt.wantVersion || !reflect.DeepEqual(rest, tt.wantRest) {
			t.Errorf("SplitArgs(%q) = (%q, %q), want (%q, %q)", tt.in, v, rest, tt.wantVersion, tt.wantRest)
		}
	}
}

func TestCheckFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "script.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(dir, "script.php")

	tests := []struct {
		name    string
		rest    []string
		wantErr string
	}{
		{name: "file", rest: []string{"script.php", "arg"}},
		{name: "absolute file", rest: []string{abs}},
		{name: "-f file", rest: []string{"-f", "script.php"}},
		{name: "--file file", rest: []string{"--file", "script.php", "--flag"}},
		{name: "no args", rest: nil, wantErr: "no PHP file given"},
		{name: "flag only", rest: []string{"-v"}, wantErr: "no PHP file given"},
		{name: "inline code", rest: []string{"-r", "echo 1;"}, wantErr: "no PHP file given"},
		{name: "--file without value", rest: []string{"--file"}, wantErr: "no PHP file given"},
		{name: "missing file", rest: []string{"nope.php"}, wantErr: "PHP file nope.php not found"},
		{name: "directory", rest: []string{"src"}, wantErr: "src is a directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFile(tt.rest, dir)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
