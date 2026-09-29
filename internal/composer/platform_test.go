package composer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirementsFor(t *testing.T) {
	tests := []struct {
		args  string
		want  Requirements
		check bool
	}{
		{"install", Requirements{Lock: true, Dev: true}, true},
		{"--ansi i --no-dev", Requirements{Lock: true}, true},
		{"update", Requirements{Dev: true}, true},
		{"u --no-dev", Requirements{}, true},
		{"require monolog/monolog", Requirements{}, false},
		{"install --ignore-platform-reqs", Requirements{}, false},
		{"install --working-dir=app", Requirements{}, false},
		{"-d app install", Requirements{}, false},
		{"", Requirements{}, false},
	}
	for _, tt := range tests {
		got, check := RequirementsFor(strings.Fields(tt.args))
		if got != tt.want || check != tt.check {
			t.Errorf("RequirementsFor(%q) = (%+v, %v), want (%+v, %v)", tt.args, got, check, tt.want, tt.check)
		}
	}
}

func TestRequiredExtensions(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("composer.json", `{
		"require": {"php": ">=8.2", "ext-intl": "*", "ext-PDO_MySQL": "*", "ext-gd": "*"},
		"require-dev": {"ext-xdebug": "*"},
		"config": {"platform": {"ext-gd": "2.0", "ext-sodium": false}}
	}`)
	write("composer.lock", `{
		"packages": [{"require": {"ext-mbstring": "*", "symfony/polyfill": "^1"}}],
		"packages-dev": [{"require": {"ext-pcov": "*"}}]
	}`)
	getenv := func(string) string { return "" }

	tests := []struct {
		r    Requirements
		want string
	}{
		{Requirements{}, "intl,pdo_mysql"},
		{Requirements{Dev: true}, "intl,pdo_mysql,xdebug"},
		{Requirements{Lock: true}, "intl,mbstring,pdo_mysql"},
		{Requirements{Lock: true, Dev: true}, "intl,mbstring,pcov,pdo_mysql,xdebug"},
	}
	for _, tt := range tests {
		if got := strings.Join(RequiredExtensions(dir, tt.r, getenv), ","); got != tt.want {
			t.Errorf("RequiredExtensions(%+v) = %s, want %s", tt.r, got, tt.want)
		}
	}

	t.Run("$COMPOSER names the file", func(t *testing.T) {
		write("other.json", `{"require": {"ext-redis": "*"}}`)
		got := RequiredExtensions(dir, Requirements{Lock: true}, func(string) string { return "other.json" })
		if strings.Join(got, ",") != "redis" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("no composer.json", func(t *testing.T) {
		if got := RequiredExtensions(t.TempDir(), Requirements{Lock: true}, getenv); got != nil {
			t.Errorf("got %v", got)
		}
	})
}
