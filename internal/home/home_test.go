package home

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// install registers version v in m, pointing at a real (empty) binary file.
func install(t *testing.T, m *Dir, v string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := m.RegisterVersion(v, bin+"\n"); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestVersionBinary(t *testing.T) {
	m := New(t.TempDir())
	bin := install(t, m, "8.3")

	got, err := m.VersionBinary("8.3")
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Errorf("VersionBinary = %q, want %q (trimmed)", got, bin)
	}

	if _, err := m.VersionBinary("7.4"); err == nil {
		t.Error("VersionBinary(7.4) expected ErrVersionNotInstalled")
	}
}

func TestVersionInstalled(t *testing.T) {
	m := New(t.TempDir())
	bin := install(t, m, "8.3")

	if !m.VersionInstalled("8.3") {
		t.Error("8.3 should be installed")
	}
	if m.VersionInstalled("8.2") {
		t.Error("8.2 should not be installed")
	}

	if err := os.Remove(bin); err != nil {
		t.Fatal(err)
	}
	if m.VersionInstalled("8.3") {
		t.Error("8.3 should not count as installed once its binary is gone")
	}
}

func TestInstalledVersions(t *testing.T) {
	t.Run("missing base dir", func(t *testing.T) {
		m := New(filepath.Join(t.TempDir(), "nope"))
		got, err := m.InstalledVersions()
		if err != nil || got != nil {
			t.Fatalf("InstalledVersions = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("sorted semantically and filtered", func(t *testing.T) {
		m := New(t.TempDir())
		for _, v := range []string{"8.10", "7.4", "8.3", "8.2"} {
			install(t, m, v)
		}
		// Not a valid version name.
		if err := os.MkdirAll(m.VersionDir("garbage"), 0755); err != nil {
			t.Fatal(err)
		}
		// Valid name but no binary metadata.
		if err := os.MkdirAll(m.VersionDir("8.1"), 0755); err != nil {
			t.Fatal(err)
		}
		// Plain file, not a directory.
		if err := os.WriteFile(filepath.Join(m.Base, "versions", "8.0"), nil, 0644); err != nil {
			t.Fatal(err)
		}

		got, err := m.InstalledVersions()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"7.4", "8.2", "8.3", "8.10"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("InstalledVersions = %v, want %v", got, want)
		}
	})
}

func TestRemoveVersionDir(t *testing.T) {
	m := New(t.TempDir())
	install(t, m, "8.3")

	if err := m.RemoveVersionDir("8.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.VersionDir("8.3")); !os.IsNotExist(err) {
		t.Errorf("version dir still exists: %v", err)
	}
}

func TestMatchInstalled(t *testing.T) {
	m := New(t.TempDir())
	for _, v := range []string{"8.2", "8.3.9", "8.3.30", "8.4.1"} {
		install(t, m, v)
	}

	tests := []struct {
		in, want string
		ok       bool
	}{
		{in: "8.2", want: "8.2", ok: true},
		{in: "8.3", want: "8.3.30", ok: true},
		{in: "8.3.9", want: "8.3.9", ok: true},
		{in: "8.4", want: "8.4.1", ok: true},
		{in: "8.3.1", ok: false},
		{in: "8.1", ok: false},
		{in: "lts", ok: false},
	}

	for _, tt := range tests {
		got, ok := m.MatchInstalled(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("MatchInstalled(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestPaths(t *testing.T) {
	d := New(filepath.Join("base"))

	shims := "shims"
	if runtime.GOOS == "linux" {
		shims = "bin"
	}
	for got, want := range map[string]string{
		d.ShimDir():       filepath.Join("base", shims),
		d.PHPDir("8.3"):   filepath.Join("base", "php", "8.3"),
		d.ComposerDir():   filepath.Join("base", "composer"),
		d.CacheDir():      filepath.Join("base", "cache"),
		d.CurrentFile():   filepath.Join("base", "current-version"),
		d.VersionDir("8"): filepath.Join("base", "versions", "8"),
	} {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}
}

func TestDefaultHonorsPVMHome(t *testing.T) {
	t.Setenv("PVM_HOME", "/custom/pvm")
	if got := Default().Base; got != "/custom/pvm" {
		t.Errorf("Default().Base = %q, want /custom/pvm", got)
	}
}
