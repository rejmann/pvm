package home

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// install registers version v in d, pointing at a real (empty) binary file.
func install(t *testing.T, d *Dir, v string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "php"+v)
	if err := os.WriteFile(bin, nil, 0755); err != nil {
		t.Fatal(err)
	}
	dir := d.VersionDir(v)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "binary"), []byte(bin+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestBinary(t *testing.T) {
	d := New(t.TempDir())
	bin := install(t, d, "8.3")

	got, err := d.Binary("8.3")
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Errorf("Binary = %q, want %q (trimmed)", got, bin)
	}

	if _, err := d.Binary("7.4"); err == nil {
		t.Error("Binary(7.4) expected ErrVersionNotInstalled")
	}
}

func TestInstalled(t *testing.T) {
	d := New(t.TempDir())
	bin := install(t, d, "8.3")

	if !d.Installed("8.3") {
		t.Error("8.3 should be installed")
	}
	if d.Installed("8.2") {
		t.Error("8.2 should not be installed")
	}

	if err := os.Remove(bin); err != nil {
		t.Fatal(err)
	}
	if d.Installed("8.3") {
		t.Error("8.3 should not count as installed once its binary is gone")
	}
}

func TestVersions(t *testing.T) {
	t.Run("missing base dir", func(t *testing.T) {
		d := New(filepath.Join(t.TempDir(), "nope"))
		got, err := d.Versions()
		if err != nil || got != nil {
			t.Fatalf("Versions = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("sorted semantically and filtered", func(t *testing.T) {
		d := New(t.TempDir())
		for _, v := range []string{"8.10", "7.4", "8.3", "8.2"} {
			install(t, d, v)
		}
		// Not a valid version name.
		if err := os.MkdirAll(d.VersionDir("garbage"), 0755); err != nil {
			t.Fatal(err)
		}
		// Valid name but no binary metadata.
		if err := os.MkdirAll(d.VersionDir("8.1"), 0755); err != nil {
			t.Fatal(err)
		}
		// Plain file, not a directory.
		if err := os.WriteFile(filepath.Join(d.Path, "versions", "8.0"), nil, 0644); err != nil {
			t.Fatal(err)
		}

		got, err := d.Versions()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"7.4", "8.2", "8.3", "8.10"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Versions = %v, want %v", got, want)
		}
	})
}

func TestRemoveVersion(t *testing.T) {
	d := New(t.TempDir())
	install(t, d, "8.3")

	if err := d.RemoveVersion("8.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(d.VersionDir("8.3")); !os.IsNotExist(err) {
		t.Errorf("version dir still exists: %v", err)
	}
}

func TestMatch(t *testing.T) {
	d := New(t.TempDir())
	for _, v := range []string{"8.2", "8.3.9", "8.3.30", "8.4.1"} {
		install(t, d, v)
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
		got, ok := d.Match(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("Match(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
