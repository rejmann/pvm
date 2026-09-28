package resolve

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/home"
)

type fakeResolver struct {
	v   string
	err error
}

func (f fakeResolver) ResolveLTS() (string, error) { return f.v, f.err }

func newHome(t *testing.T) *home.Dir {
	t.Helper()
	h := home.New(t.TempDir())
	if err := h.EnsureBaseDir(); err != nil {
		t.Fatal(err)
	}
	return h
}

// fakeInstall registers version v in h, pointing at an existing executable.
func fakeInstall(t *testing.T, h *home.Dir, v string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := h.RegisterVersion(v, bin); err != nil {
		t.Fatal(err)
	}
}

func writePHPVersion(t *testing.T, dir, v string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".php-version"), []byte(v+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestActive(t *testing.T) {
	h := newHome(t)
	for _, v := range []string{"7.4", "8.2", "8.3.30"} {
		fakeInstall(t, h, v)
	}
	if err := h.SetCurrent("7.4"); err != nil {
		t.Fatal(err)
	}

	project := t.TempDir()
	writePHPVersion(t, project, "8.2")
	nested := filepath.Join(project, "src", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

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
			a, err := Active(h, tt.dir, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			if a.Version != tt.wantVersion || a.Source != tt.wantSource {
				t.Errorf("Active = {%s, %s}, want {%s, %s}", a.Version, a.Source, tt.wantVersion, tt.wantSource)
			}
			wantBin, _ := h.VersionBinary(tt.wantVersion)
			if a.Binary != wantBin {
				t.Errorf("Binary = %q, want %q", a.Binary, wantBin)
			}
		})
	}
}

func TestActiveErrors(t *testing.T) {
	t.Run("nothing selected", func(t *testing.T) {
		h := newHome(t)
		if _, err := Active(h, t.TempDir(), ""); !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})

	t.Run("project version not installed", func(t *testing.T) {
		h := newHome(t)
		fakeInstall(t, h, "8.3")
		dir := t.TempDir()
		writePHPVersion(t, dir, "8.1")

		_, err := Active(h, dir, "")
		if err == nil || !strings.Contains(err.Error(), "PHP 8.1 (set by "+filepath.Join(dir, ".php-version")+") is not installed") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestVersion(t *testing.T) {
	h := newHome(t)
	fakeInstall(t, h, "8.2")
	fakeInstall(t, h, "8.5.1")

	tests := []struct {
		name          string
		arg           string
		r             fakeResolver
		wantInstalled string
		wantErr       string
	}{
		{name: "exact version", arg: "8.2", wantInstalled: "8.2"},
		{name: "branch matches installed patch", arg: "8.5", wantInstalled: "8.5.1"},
		{name: "lts alias", arg: "lts", r: fakeResolver{v: "8.5"}, wantInstalled: "8.5.1"},
		{name: "not installed", arg: "8.1", wantErr: "PHP 8.1 is not installed — run: pvm install 8.1"},
		{name: "invalid version", arg: "8.x", wantErr: "invalid version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Version(h, tt.arg, tt.r)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantBin, _ := h.VersionBinary(tt.wantInstalled)
			if got.Version != tt.wantInstalled || got.Binary != wantBin {
				t.Errorf("Version = (%q, %q), want (%q, %q)", got.Version, got.Binary, tt.wantInstalled, wantBin)
			}
		})
	}

	t.Run("resolver error", func(t *testing.T) {
		boom := errors.New("offline")
		if _, err := Version(h, "lts", fakeResolver{err: boom}); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}
