package pvm

import (
	"errors"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/home"
	"github.com/rejmann/pvm/internal/sysphp"
)

type fakeExtManager struct {
	fakeExtensions
	removed []string
	toggled []string // "ext=on" or "ext=off"
}

func (f *fakeExtManager) RemoveExtensions(_ *home.Dir, _ string, exts []string) error {
	f.removed = append(f.removed, exts...)
	return f.err
}

func (f *fakeExtManager) SetExtensionsEnabled(_ *home.Dir, _ string, exts []string, enabled bool) error {
	state := "off"
	if enabled {
		state = "on"
	}
	for _, ext := range exts {
		f.toggled = append(f.toggled, ext+"="+state)
	}
	return f.err
}

// newTestExtensions manages the extensions of PHP 8.4 (the global version),
// whose php loads loaded.
func newTestExtensions(t *testing.T, loaded ...string) (*Extensions, *fakeExtManager) {
	t.Helper()
	m, _, _ := newTestManager(t)
	fakeInstall(t, m.Home, "8.4")
	setGlobal(t, m.Home, "8.4")
	inst := &fakeExtManager{}
	return &Extensions{
		Manager:   m,
		Installer: inst,
		Probe:     func(string) (sysphp.Info, error) { return sysphp.Info{Version: "8.4.1", Extensions: loaded}, nil },
	}, inst
}

func TestExtensionsVersion(t *testing.T) {
	e, _ := newTestExtensions(t)
	if a, err := e.Version("", t.TempDir(), "", false); err != nil || a.Version != "8.4" {
		t.Fatalf("Version = (%+v, %v)", a, err)
	}

	fakeInstall(t, e.Manager.Home, "8.2")
	bin, _ := e.Manager.Home.Binary("8.2")
	if err := e.Manager.Home.SetSystem("8.2", bin); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Version("8.2", "", "", false); err == nil || !strings.Contains(err.Error(), "installed outside pvm") {
		t.Errorf("error = %v, want version installed outside pvm refused", err)
	}
	if a, err := e.Version("8.2", "", "", true); err != nil || !a.System {
		t.Errorf("read-only Version = (%+v, %v)", a, err)
	}
}

func TestExtensionsAdd(t *testing.T) {
	e, inst := newTestExtensions(t, "intl", "opcache")
	a, _ := e.Version("", t.TempDir(), "", false)

	added, loaded, err := e.Add(a, []string{"ext-intl", "Redis", "redis", "Zend OPcache", "xdebug"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(added, ",") != "redis,xdebug" || strings.Join(loaded, ",") != "intl,opcache" ||
		strings.Join(inst.added, ",") != "redis,xdebug" {
		t.Errorf("added = %v, loaded = %v, installed = %v", added, loaded, inst.added)
	}

	t.Run("all loaded: nothing to install", func(t *testing.T) {
		e, inst := newTestExtensions(t, "intl")
		a, _ := e.Version("", t.TempDir(), "", false)
		if added, _, err := e.Add(a, []string{"intl"}); err != nil || added != nil || inst.added != nil {
			t.Errorf("Add = (%v, %v), installed = %v", added, err, inst.added)
		}
	})

	t.Run("install fails", func(t *testing.T) {
		e, inst := newTestExtensions(t)
		inst.err = errors.New("boom")
		a, _ := e.Version("", t.TempDir(), "", false)
		if added, _, err := e.Add(a, []string{"redis"}); err == nil || added != nil {
			t.Errorf("Add = (%v, %v), want error", added, err)
		}
	})
}

func TestExtensionsRemoveAndToggle(t *testing.T) {
	e, inst := newTestExtensions(t)
	a, _ := e.Version("", t.TempDir(), "", false)

	if err := e.Remove(a, []string{"ext-Redis", "redis"}); err != nil {
		t.Fatal(err)
	}
	if err := e.SetEnabled(a, []string{"Xdebug"}, false); err != nil {
		t.Fatal(err)
	}
	if err := e.SetEnabled(a, []string{"xdebug"}, true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(inst.removed, ",") != "redis" || strings.Join(inst.toggled, ",") != "xdebug=off,xdebug=on" {
		t.Errorf("removed = %v, toggled = %v", inst.removed, inst.toggled)
	}
}
