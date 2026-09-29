package installer

import "testing"

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
