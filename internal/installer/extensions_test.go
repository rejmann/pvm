package installer

import (
	"strings"
	"testing"
)

func TestNormalizeExtension(t *testing.T) {
	for in, want := range map[string]string{
		"ext-xml":       "xml",
		"EXT-DOM":       "xml",
		"simplexml":     "xml",
		"ext-pdo_mysql": "mysql",
		"pdo_sqlite":    "sqlite3",
		" intl ":        "intl",
	} {
		if got := normalizeExtension(in); got != want {
			t.Errorf("normalizeExtension(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecordPackages(t *testing.T) {
	base := t.TempDir()
	if got := readPackages(base, "8.4"); got != nil {
		t.Fatalf("readPackages without file = %v", got)
	}
	if err := recordPackages(base, "8.4", []string{"php8.4-zip", "php8.4-xml"}); err != nil {
		t.Fatal(err)
	}
	if err := recordPackages(base, "8.4", []string{"php8.4-xml", "php8.4-intl"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(readPackages(base, "8.4"), ","); got != "php8.4-zip,php8.4-xml,php8.4-intl" {
		t.Errorf("packages = %s", got)
	}
	if got := readPackages(base, "8.3"); got != nil {
		t.Errorf("other version has packages %v", got)
	}
}
