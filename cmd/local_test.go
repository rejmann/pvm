package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetLocal(t *testing.T) {
	t.Run("writes .php-version", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.2")
		dir := t.TempDir()
		var out bytes.Buffer

		if err := setLocal("8.2", m, failResolver{t}, dir, &out); err != nil {
			t.Fatal(err)
		}
		assertFile(t, filepath.Join(dir, ".php-version"), "8.2\n")
		if !strings.Contains(out.String(), "PHP 8.2 will be used in "+dir) {
			t.Errorf("unexpected output: %q", out.String())
		}
	})

	t.Run("resolves lts to a concrete version", func(t *testing.T) {
		m := newManager(t)
		fakeInstall(t, m, "8.4")
		dir := t.TempDir()

		if err := setLocal("lts", m, fakeResolver{v: "8.4"}, dir, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		assertFile(t, filepath.Join(dir, ".php-version"), "8.4\n")
	})

	t.Run("rejects version not installed", func(t *testing.T) {
		m := newManager(t)
		dir := t.TempDir()

		err := setLocal("8.1", m, failResolver{t}, dir, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("error = %v, want not installed", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".php-version")); !os.IsNotExist(err) {
			t.Error(".php-version must not be written for a missing version")
		}
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		m := newManager(t)
		err := setLocal("8.x", m, failResolver{t}, t.TempDir(), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Fatalf("error = %v, want invalid version", err)
		}
	})
}

func TestShowAndUnsetLocal(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub")
	mkdirAll(t, nested)
	var out bytes.Buffer

	if err := showLocal(nested, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "No .php-version found") {
		t.Errorf("unexpected output: %q", out.String())
	}

	writePHPVersion(t, dir, "8.3")
	out.Reset()
	if err := showLocal(nested, &out); err != nil {
		t.Fatal(err)
	}
	if want := "8.3 (set by " + filepath.Join(dir, ".php-version") + ")\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}

	if err := unsetLocal(dir, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".php-version")); !os.IsNotExist(err) {
		t.Error(".php-version should have been removed")
	}
	// Unsetting again is a no-op.
	if err := unsetLocal(dir, &bytes.Buffer{}); err != nil {
		t.Errorf("second unset = %v, want nil", err)
	}
}

func TestUseArg(t *testing.T) {
	dir := t.TempDir()

	got, err := useArg([]string{"8.1"}, dir, &bytes.Buffer{})
	if err != nil || got != "8.1" {
		t.Fatalf("explicit arg: (%q, %v)", got, err)
	}

	if _, err := useArg(nil, dir, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "no .php-version found") {
		t.Fatalf("no file: error = %v", err)
	}

	writePHPVersion(t, dir, "8.2")
	var out bytes.Buffer
	got, err = useArg(nil, dir, &out)
	if err != nil || got != "8.2" {
		t.Fatalf("from file: (%q, %v)", got, err)
	}
	if !strings.Contains(out.String(), "with version 8.2") {
		t.Errorf("unexpected output: %q", out.String())
	}
}

func TestPrintWhich(t *testing.T) {
	m := newManager(t)
	fakeInstall(t, m, "8.3")
	dir := t.TempDir()
	writePHPVersion(t, dir, "8.3")
	var out bytes.Buffer

	if err := printWhich(m, dir, "", &out); err != nil {
		t.Fatal(err)
	}
	bin, _ := m.GetVersionBinary("8.3")
	if out.String() != bin+"\n" {
		t.Errorf("output = %q, want %q", out.String(), bin+"\n")
	}

	if err := printWhich(newManager(t), t.TempDir(), "", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "pvm local") {
		t.Errorf("nothing selected: error = %v", err)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
