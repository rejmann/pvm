package pvm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/composer"
	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/sysphp"
)

type fakeSource struct{ phar string }

func (f fakeSource) Ensure(_ context.Context, _, _, _ string, onDownload func(composer.Release)) (string, error) {
	onDownload(composer.Release{Version: "2.10.3"})
	return f.phar, nil
}

type fakeExtensions struct {
	added []string
	err   error
}

func (f *fakeExtensions) AddExtensions(_ *home.Dir, _ string, exts []string) error {
	f.added = append(f.added, exts...)
	return f.err
}

// newTestComposer runs Composer for PHP 8.5 (the global version) with fakes;
// runs[i] is what the i-th Composer run prints on stderr and its exit code.
func newTestComposer(t *testing.T, runs ...fakeRun) (*Composer, *fakeExtensions, *[]fakeCall) {
	t.Helper()
	m, _, _ := newTestManager(t)
	fakeInstall(t, m.Home, "8.5")
	setGlobal(t, m.Home, "8.5")

	ext := &fakeExtensions{}
	var calls []fakeCall
	c := &Composer{
		Manager:    m,
		Source:     fakeSource{phar: "/pvm/composer.phar"},
		Extensions: ext,
		Probe:      func(string) (sysphp.Info, error) { return sysphp.Info{Version: "8.5.1", Zip: true}, nil },
		Exec: func(bin string, args []string, env map[string]string, watch io.Writer) (int, error) {
			i := len(calls)
			calls = append(calls, fakeCall{args: args, env: env})
			if i >= len(runs) {
				t.Fatalf("unexpected Composer run %d", i+1)
			}
			io.WriteString(watch, runs[i].stderr)
			return runs[i].code, nil
		},
		Confirm: func(string) bool { return true },
		Getenv:  func(string) string { return "" },
		Notices: io.Discard,
	}
	return c, ext, &calls
}

type fakeRun struct {
	stderr string
	code   int
}

type fakeCall struct {
	args []string
	env  map[string]string
}

func TestComposerRun(t *testing.T) {
	t.Run("runs the phar with the version's environment", func(t *testing.T) {
		c, ext, calls := newTestComposer(t, fakeRun{code: 3})

		code, err := c.Run(context.Background(), t.TempDir(), []string{"install"})
		if err != nil || code != 3 {
			t.Fatalf("Run = (%d, %v), want (3, nil)", code, err)
		}
		call := (*calls)[0]
		if strings.Join(call.args, " ") != "/pvm/composer.phar install" {
			t.Errorf("args = %v", call.args)
		}
		if call.env[EnvVersion] != "8.5" || call.env["COMPOSER_HOME"] == "" {
			t.Errorf("env = %v", call.env)
		}
		if ext.added != nil {
			t.Errorf("installed extensions %v", ext.added)
		}
	})

	t.Run("installs missing extensions and runs again", func(t *testing.T) {
		project := filepath.Join(t.TempDir(), "app")
		mkdirAll(t, filepath.Join(project, "vendor"))
		c, ext, calls := newTestComposer(t,
			fakeRun{stderr: "Created project in " + project + "\n" +
				"  - symfony/framework-bundle v8.1.7 requires ext-xml * -> it is missing from your system.\n"},
			fakeRun{code: 0},
		)

		code, err := c.Run(context.Background(), t.TempDir(), []string{"create-project", "symfony/skeleton", "app"})
		if err != nil || code != 0 {
			t.Fatalf("Run = (%d, %v)", code, err)
		}
		if strings.Join(ext.added, ",") != "xml" || len(*calls) != 2 {
			t.Errorf("added = %v, runs = %d", ext.added, len(*calls))
		}
		if entries, _ := os.ReadDir(project); len(entries) != 0 {
			t.Errorf("project dir not emptied before the second run: %v", entries)
		}
	})

	t.Run("declined: no second run", func(t *testing.T) {
		c, ext, calls := newTestComposer(t, fakeRun{stderr: "x requires ext-intl * -> it is missing from your system.\n", code: 2})
		c.Confirm = func(string) bool { return false }

		if code, err := c.Run(context.Background(), t.TempDir(), []string{"install"}); err != nil || code != 2 {
			t.Fatalf("Run = (%d, %v), want (2, nil)", code, err)
		}
		if ext.added != nil || len(*calls) != 1 {
			t.Errorf("added = %v, runs = %d", ext.added, len(*calls))
		}
	})

	t.Run("offers zip up front", func(t *testing.T) {
		c, ext, _ := newTestComposer(t, fakeRun{})
		c.Probe = func(string) (sysphp.Info, error) { return sysphp.Info{Version: "8.5.1"}, nil }

		if _, err := c.Run(context.Background(), t.TempDir(), nil); err != nil {
			t.Fatal(err)
		}
		if strings.Join(ext.added, ",") != "zip" {
			t.Errorf("added = %v, want zip", ext.added)
		}
	})

	t.Run("no version selected", func(t *testing.T) {
		c, _, _ := newTestComposer(t)
		if err := c.Manager.Home.ClearCurrent(); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Run(context.Background(), t.TempDir(), nil); !errors.Is(err, ErrNoActiveVersion) {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})
}

func TestOfferExtensions(t *testing.T) {
	t.Run("not a terminal", func(t *testing.T) {
		c, ext, _ := newTestComposer(t)
		var out bytes.Buffer
		c.Confirm, c.Notices = nil, &out

		if c.offerExtensions("8.3", []string{"zip"}, "which Composer needs") {
			t.Error("reported installed")
		}
		if !strings.Contains(out.String(), "PHP 8.3 is missing the zip extension, which Composer needs") ||
			!strings.Contains(out.String(), "in a terminal") || ext.added != nil {
			t.Errorf("output = %q, added = %v", out.String(), ext.added)
		}
	})

	t.Run("accepted", func(t *testing.T) {
		c, ext, _ := newTestComposer(t)
		var out bytes.Buffer
		c.Notices = &out

		if !c.offerExtensions("8.5", []string{"xml", "intl"}, "x") {
			t.Errorf("not installed, output = %q", out.String())
		}
		if strings.Join(ext.added, ",") != "xml,intl" || !strings.Contains(out.String(), "the xml, intl extensions") {
			t.Errorf("added %v, output = %q", ext.added, out.String())
		}
	})

	t.Run("install fails", func(t *testing.T) {
		c, ext, _ := newTestComposer(t)
		var out bytes.Buffer
		c.Notices = &out
		ext.err = errors.New("boom")

		if c.offerExtensions("8.5", []string{"xml"}, "x") {
			t.Error("reported installed")
		}
		if !strings.Contains(out.String(), "Warning: boom") {
			t.Errorf("output = %q", out.String())
		}
	})
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
