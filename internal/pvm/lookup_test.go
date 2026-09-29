package pvm

import (
	"errors"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	m, _, _ := newTestManager(t)
	h := m.Home
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
			m.LTS = tt.r
			a, err := m.Lookup(tt.arg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantBin, _ := h.Binary(tt.wantInstalled)
			if a.Version != tt.wantInstalled || a.Binary != wantBin {
				t.Errorf("Lookup = (%q, %q), want (%q, %q)", a.Version, a.Binary, tt.wantInstalled, wantBin)
			}
		})
	}

	t.Run("resolver error", func(t *testing.T) {
		boom := errors.New("offline")
		m.LTS = fakeResolver{err: boom}
		if _, err := m.Lookup("lts"); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}

func TestSelect(t *testing.T) {
	m, _, _ := newTestManager(t)
	h := m.Home
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
			a, err := m.Select(tt.versionArg, tt.dir, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			wantBin, _ := h.Binary(tt.want)
			if a.Version != tt.want || a.Binary != wantBin {
				t.Errorf("Select = (%q, %q), want (%q, %q)", a.Version, a.Binary, tt.want, wantBin)
			}
		})
	}

	t.Run("nothing selected", func(t *testing.T) {
		empty, _, _ := newTestManager(t)
		if _, err := empty.Select("", t.TempDir(), ""); !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v", err)
		}
	})
}
