package composer

import (
	"strings"
	"testing"
)

func TestComposerOutput(t *testing.T) {
	stderr := "Creating a \"symfony/skeleton:8.1.*\" project at \"./app\"\n" +
		"\x1b[32mCreated project in /work/app\x1b[39m\r\n" +
		"Your requirements could not be resolved to an installable set of packages.\n\n" +
		"  Problem 1\n" +
		"    - Root composer.json requires PHP extension ext-intl * but it is missing from your system. Install or enable PHP's intl extension.\n" +
		"  Problem 2\n" +
		"    - \x1b[32msymfony/framework-bundle\x1b[39m[v8.1.0, ..., v8.1.7] require \x1b[37mext-xml\x1b[39m * -> it is missing from your system.\n" +
		"    - symfony/config v8.1.5 requires ext-xml * -> it is missing from your system.\n" +
		"Alternatively, you can run Composer with `--ignore-platform-req=ext-mbstring` to temporarily ignore these required extensions.\n" +
		"    - x/y requires ext-gd * -> it is missing from your system." // no final newline

	o := &Output{}
	// Written in small chunks, as a pipe delivers it.
	for i := 0; i < len(stderr); i += 7 {
		o.Write([]byte(stderr[i:min(i+7, len(stderr))]))
	}
	o.Flush()

	if got := strings.Join(o.Missing, ","); got != "intl,xml,gd" {
		t.Errorf("missing = %s, want intl,xml,gd", got)
	}
	if o.Project != "/work/app" {
		t.Errorf("project = %q, want /work/app", o.Project)
	}

	clean := &Output{}
	clean.Write([]byte("Could not find package foo/bar.\n"))
	if clean.Missing != nil || clean.Project != "" {
		t.Errorf("clean output = %+v", clean)
	}
}
