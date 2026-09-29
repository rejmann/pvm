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
