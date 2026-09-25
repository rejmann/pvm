package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveActive(t *testing.T) {
	m := newManager(t)
	for _, v := range []string{"7.4", "8.2", "8.3.30"} {
		fakeInstall(t, m, v)
	}
	setGlobal(t, m.Base, "7.4")

	project := t.TempDir()
	writePHPVersion(t, project, "8.2")
	nested := filepath.Join(project, "src", "app")
	mkdirAll(t, nested)

	tests := []struct {
		name        string
		dir, env    string
		wantVersion string
		wantSource  string
	}{
		{name: "global fallback", dir: t.TempDir(), wantVersion: "7.4", wantSource: "global"},
		{name: "project file", dir: project, wantVersion: "8.2", wantSource: filepath.Join(project, ".php-version")},
		{name: "project file from subdirectory", dir: nested, wantVersion: "8.2", wantSource: filepath.Join(project, ".php-version")},
		{name: "env overrides project file", dir: project, env: "7.4", wantVersion: "7.4", wantSource: "PVM_VERSION environment variable"},
		{name: "branch matches installed patch", dir: t.TempDir(), env: "8.3", wantVersion: "8.3.30", wantSource: "PVM_VERSION environment variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := resolveActive(m, tt.dir, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			if a.Version != tt.wantVersion || a.Source != tt.wantSource {
				t.Errorf("resolveActive = {%s, %s}, want {%s, %s}", a.Version, a.Source, tt.wantVersion, tt.wantSource)
			}
			wantBin, _ := m.GetVersionBinary(tt.wantVersion)
			if a.Binary != wantBin {
				t.Errorf("Binary = %q, want %q", a.Binary, wantBin)
			}
		})
	}
}

func TestResolveActiveErrors(t *testing.T) {
	t.Run("nothing selected", func(t *testing.T) {
		m := newManager(t)
		if _, err := resolveActive(m, t.TempDir(), ""); !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})

	t.Run("project version not installed", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.3")
		dir := t.TempDir()
		writePHPVersion(t, dir, "8.1")

		_, err := resolveActive(m, dir, "")
		if err == nil || !strings.Contains(err.Error(), "PHP 8.1 (set by "+filepath.Join(dir, ".php-version")+") is not installed") {
			t.Fatalf("error = %v", err)
		}
	})
}
