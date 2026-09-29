//go:build linux

package installer

import "testing"

func TestRemiExtPkg(t *testing.T) {
	for ext, want := range map[string]string{
		"xml":     "php8.4-php-xml",
		"zip":     "php8.4-php-pecl-zip",
		"mysql":   "php8.4-php-mysqlnd",
		"sqlite3": "php8.4-php-pdo",
	} {
		if got := remiExtPkg("8.4", ext); got != want {
			t.Errorf("remiExtPkg(%q) = %q, want %q", ext, got, want)
		}
	}
}
