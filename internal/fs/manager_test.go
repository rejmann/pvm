package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// install registers version v in m, pointing at a real (empty) binary file.
func install(t *testing.T, m *Manager, v string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	dir := m.VersionDir(v)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "binary"), []byte(bin+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestGetVersionBinary(t *testing.T) {
	m := NewManager(t.TempDir())
	bin := install(t, m, "8.3")

	got, err := m.GetVersionBinary("8.3")
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Errorf("GetVersionBinary = %q, want %q (trimmed)", got, bin)
	}

	if _, err := m.GetVersionBinary("7.4"); err == nil {
		t.Error("GetVersionBinary(7.4) expected ErrVersionNotInstalled")
	}
}

func TestVersionInstalled(t *testing.T) {
	m := NewManager(t.TempDir())
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
		m := NewManager(filepath.Join(t.TempDir(), "nope"))
		got, err := m.InstalledVersions()
		if err != nil || got != nil {
			t.Fatalf("InstalledVersions = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("sorted semantically and filtered", func(t *testing.T) {
		m := NewManager(t.TempDir())
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
	m := NewManager(t.TempDir())
	install(t, m, "8.3")

	if err := m.RemoveVersionDir("8.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.VersionDir("8.3")); !os.IsNotExist(err) {
		t.Errorf("version dir still exists: %v", err)
	}
}
