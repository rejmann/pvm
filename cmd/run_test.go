package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/resolve"
)

func TestSplitRunArgs(t *testing.T) {
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
		v, rest, err := splitRunArgs(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("splitRunArgs(%q) error = %v, want %q", tt.in, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitRunArgs(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if v != tt.wantVersion || !reflect.DeepEqual(rest, tt.wantRest) {
			t.Errorf("splitRunArgs(%q) = (%q, %q), want (%q, %q)", tt.in, v, rest, tt.wantVersion, tt.wantRest)
		}
	}
}

func TestRunResolveUsesActiveVersion(t *testing.T) {
	h := newHome(t)
	fakeInstall(t, h, "8.2")
	fakeInstall(t, h, "8.5")
	setGlobal(t, h, "8.5")
	project := t.TempDir()
	writePHPVersion(t, project, "8.2")

	tests := []struct {
		name, versionArg, dir, env, want string
	}{
		{name: "global", dir: t.TempDir(), want: "8.5"},
		{name: "project .php-version", dir: project, want: "8.2"},
		{name: "PVM_VERSION", dir: project, env: "8.5", want: "8.5"},
		{name: "explicit version wins", versionArg: "8.2", dir: t.TempDir(), env: "8.5", want: "8.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runResolve(tt.versionArg, h, failResolver{t}, tt.dir, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			wantBin, _ := h.VersionBinary(tt.want)
			if got.Version != tt.want || got.Binary != wantBin {
				t.Errorf("runResolve = (%q, %q), want (%q, %q)", got.Version, got.Binary, tt.want, wantBin)
			}
		})
	}

	t.Run("nothing selected", func(t *testing.T) {
		_, err := runResolve("", newHome(t), failResolver{t}, t.TempDir(), "")
		if !errors.Is(err, resolve.ErrNoActiveVersion) || !strings.Contains(err.Error(), "pvm run 8.3") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestCheckRunFile(t *testing.T) {
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, "src"))
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
			err := checkRunFile(tt.rest, dir)
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
