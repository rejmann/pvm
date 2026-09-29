package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/home"
)

func TestParseProbe(t *testing.T) {
	tests := []struct {
		out     string
		want    phpInfo
		wantErr bool
	}{
		{out: "\n8.3.12\n1", want: phpInfo{Version: "8.3.12", Zip: true}},
		{out: "\n7.4.33\n0\n", want: phpInfo{Version: "7.4.33"}},
		{out: "PHP Warning:  Module \"x\" is already loaded\n\n8.5.0\r\n1", want: phpInfo{Version: "8.5.0", Zip: true}},
		{out: "8.3.12", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseProbe(tt.out)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseProbe(%q) = %+v, want error", tt.out, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("parseProbe(%q) = (%+v, %v), want %+v", tt.out, got, err, tt.want)
		}
	}
}

func TestHasArchiveTool(t *testing.T) {
	if hasArchiveTool(t.TempDir()) {
		t.Error("found an archive tool in an empty PATH")
	}
	for _, name := range []string{"unzip", "7z"} {
		sys := fakeExecutable(t, t.TempDir(), name)
		if !hasArchiveTool(filepath.Dir(sys)) {
			t.Errorf("%s on PATH not found", name)
		}
	}
}

func TestOfferExtensions(t *testing.T) {
	var got []string
	saved := installExtensions
	installExtensions = func(_ *home.Dir, _ string, exts []string) error {
		got = append(got, exts...)
		return nil
	}
	t.Cleanup(func() { installExtensions = saved })

	t.Run("not a terminal", func(t *testing.T) {
		var out bytes.Buffer
		if offerExtensions(home.New(t.TempDir()), "8.3", []string{"zip"}, "which Composer needs", false, strings.NewReader("y\n"), &out) {
			t.Error("reported installed")
		}
		if !strings.Contains(out.String(), "PHP 8.3 is missing the zip extension, which Composer needs") ||
			!strings.Contains(out.String(), "in a terminal") {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("declined", func(t *testing.T) {
		var out bytes.Buffer
		if offerExtensions(home.New(t.TempDir()), "8.3", []string{"zip"}, "x", true, strings.NewReader("n\n"), &out) {
			t.Error("reported installed")
		}
	})

	t.Run("accepted", func(t *testing.T) {
		got = nil
		var out bytes.Buffer
		if !offerExtensions(home.New(t.TempDir()), "8.5", []string{"xml", "intl"}, "x", true, strings.NewReader("y\n"), &out) {
			t.Errorf("not installed, output = %q", out.String())
		}
		if strings.Join(got, ",") != "xml,intl" || !strings.Contains(out.String(), "the xml, intl extensions") {
			t.Errorf("installed %v, output = %q", got, out.String())
		}
	})

	t.Run("install fails", func(t *testing.T) {
		installExtensions = func(_ *home.Dir, _ string, _ []string) error { return errors.New("boom") }
		var out bytes.Buffer
		if offerExtensions(home.New(t.TempDir()), "8.5", []string{"xml"}, "x", true, strings.NewReader("y\n"), &out) {
			t.Error("reported installed")
		}
		if !strings.Contains(out.String(), "Warning: boom") {
			t.Errorf("output = %q", out.String())
		}
	})
}

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

	o := &composerOutput{}
	// Written in small chunks, as a pipe delivers it.
	for i := 0; i < len(stderr); i += 7 {
		o.Write([]byte(stderr[i:min(i+7, len(stderr))]))
	}
	o.Close()

	if got := strings.Join(o.missing, ","); got != "intl,xml,gd" {
		t.Errorf("missing = %s, want intl,xml,gd", got)
	}
	if o.project != "/work/app" {
		t.Errorf("project = %q, want /work/app", o.project)
	}

	clean := &composerOutput{}
	clean.Write([]byte("Could not find package foo/bar.\n"))
	if clean.missing != nil || clean.project != "" {
		t.Errorf("clean output = %+v", clean)
	}
}

func TestEmptyDir(t *testing.T) {
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, "vendor", "x"))
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := emptyDir(dir); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("dir not emptied: %v, %v", entries, err)
	}
	if err := emptyDir(filepath.Join(dir, "missing")); err != nil {
		t.Errorf("missing dir: %v", err)
	}
}

func TestForceANSI(t *testing.T) {
	tests := []struct {
		args     []string
		terminal bool
		noColor  string
		want     bool
	}{
		{args: []string{"install"}, terminal: true, want: true},
		{args: []string{"install"}, terminal: false},
		{args: []string{"install"}, terminal: true, noColor: "1"},
		{args: []string{"--no-ansi", "install"}, terminal: true},
		{args: []string{"install", "--ansi"}, terminal: true},
		{args: []string{"run", "x", "--", "--no-ansi"}, terminal: true, want: true},
	}
	for _, tt := range tests {
		if got := forceANSI(tt.args, tt.terminal, tt.noColor); got != tt.want {
			t.Errorf("forceANSI(%v, %v, %q) = %v, want %v", tt.args, tt.terminal, tt.noColor, got, tt.want)
		}
	}
}
