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

func TestEnableIniExtensions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "php.ini"), "extension=zip\n;extension=intl\n")
	for _, dll := range []string{"php_zip.dll", "php_intl.dll", "php_mbstring.dll"} {
		writeFile(t, filepath.Join(dir, "ext", dll), "")
	}

	err := enableIniExtensions(dir, []string{"ext-intl", "zip", "mbstring", "intl", "redis"})
	if err == nil || !strings.Contains(err.Error(), "no DLL for redis") {
		t.Errorf("error = %v, want redis reported", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "php.ini"))
	ini := string(data)
	if strings.Count(ini, "extension=php_intl.dll") != 1 || !strings.Contains(ini, "extension=php_mbstring.dll") {
		t.Errorf("php.ini = %q", ini)
	}
	if strings.Contains(ini, "php_zip.dll") {
		t.Errorf("zip was already enabled, php.ini = %q", ini)
	}
}
