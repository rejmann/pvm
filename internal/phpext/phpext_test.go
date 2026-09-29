package phpext

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"ext-xml":       "xml",
		"EXT-DOM":       "xml",
		"simplexml":     "xml",
		"ext-pdo_mysql": "mysql",
		"pdo_sqlite":    "sqlite3",
		" intl ":        "intl",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestName(t *testing.T) {
	for in, want := range map[string]string{
		"ext-PDO_MySQL": "pdo_mysql",
		"dom":           "dom",
		"Zend OPcache":  "opcache",
		" Xdebug ":      "xdebug",
	} {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}
