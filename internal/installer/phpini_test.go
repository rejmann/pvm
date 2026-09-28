package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestWritePHPIni(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "php.ini-production"), "memory_limit = 128M\n;extension=openssl\n")
	writeFile(t, filepath.Join(dir, "ext", "php_openssl.dll"), "")

	if err := writePHPIni(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "php.ini"))
	if err != nil {
		t.Fatal(err)
	}
	ini := string(data)

	for _, want := range []string{
		"memory_limit = 128M",
		`extension_dir = "` + filepath.Join(dir, "ext") + `"`,
		"\nextension=php_openssl.dll",
	} {
		if !strings.Contains(ini, want) {
			t.Errorf("php.ini missing %q:\n%s", want, ini)
		}
	}
	if strings.Contains(ini, "php_zip.dll") {
		t.Errorf("php.ini enables zip, whose DLL is not shipped:\n%s", ini)
	}
	if strings.Index(ini, iniMarker) < strings.Index(ini, "memory_limit") {
		t.Errorf("pvm block must come after the template:\n%s", ini)
	}
}

func TestWritePHPIniKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "php.ini"), "user edits\n")

	if err := writePHPIni(dir); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "php.ini")); string(data) != "user edits\n" {
		t.Errorf("php.ini overwritten: %q", data)
	}
}

func TestWritePHPIniWithoutTemplate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ext", "php_zip.dll"), "")

	if err := writePHPIni(dir); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "php.ini"))
	if !strings.Contains(string(data), "extension=php_zip.dll") {
		t.Errorf("php.ini = %q", data)
	}
}
