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
	if !strings.HasPrefix(ini, "extension=zip\nextension=intl\n") || strings.Contains(ini, "php_intl.dll") ||
		!strings.Contains(ini, "extension=php_mbstring.dll") {
		t.Errorf("php.ini = %q, want intl uncommented and mbstring appended", ini)
	}
	if strings.Contains(ini, "php_zip.dll") {
		t.Errorf("zip was already enabled, php.ini = %q", ini)
	}
}

func TestIniExtension(t *testing.T) {
	for line, want := range map[string]string{
		"extension=php_zip.dll":                     "zip",
		` extension = "C:\\php\\ext\\php_intl.dll"`: "intl",
		";zend_extension=opcache":                   "opcache",
		"zend_extension=/usr/lib/xdebug.so":         "xdebug",
		"extension_dir = ext":                       "",
		"; extension=... see below":                 "",
	} {
		name, _, ok := iniExtension(line)
		if name != want || ok != (want != "") {
			t.Errorf("iniExtension(%q) = (%q, %v), want %q", line, name, ok, want)
		}
	}
}

func TestDisableIniExtensions(t *testing.T) {
	ini := "extension=php_zip.dll\nzend_extension=xdebug\n;extension=intl\n"

	t.Run("comments out", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "php.ini"), ini)
		writeFile(t, filepath.Join(dir, "ext", "php_xdebug.dll"), "")

		if missing, err := disableIniExtensions(dir, []string{"xdebug"}); err != nil || missing != nil {
			t.Fatalf("disableIniExtensions = (%v, %v)", missing, err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "php.ini"))
		if got := string(data); got != "extension=php_zip.dll\n;zend_extension=xdebug\n;extension=intl\n" {
			t.Errorf("php.ini = %q", got)
		}

		// Enabling it again uncomments the same line.
		if err := enableIniExtensions(dir, []string{"xdebug"}); err != nil {
			t.Fatal(err)
		}
		data, _ = os.ReadFile(filepath.Join(dir, "php.ini"))
		if got := string(data); got != ini {
			t.Errorf("php.ini after enable = %q", got)
		}
	})

	t.Run("reports what php.ini has no line for", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "php.ini"), ini)

		// intl is already commented out: nothing to do, but not missing.
		missing, err := disableIniExtensions(dir, []string{"ext-zip", "intl", "calendar"})
		if err != nil || strings.Join(missing, ",") != "calendar" {
			t.Errorf("disableIniExtensions = (%v, %v), want calendar missing", missing, err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "php.ini"))
		if got := string(data); got != ";extension=php_zip.dll\nzend_extension=xdebug\n;extension=intl\n" {
			t.Errorf("php.ini = %q", got)
		}
	})
}
