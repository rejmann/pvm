package pvm

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/sysphp"
)

// withSystemPHP makes m find a php installed before pvm, and counts lookups.
func withSystemPHP(t *testing.T, m *Manager, v string) (bin string, lookups *int) {
	t.Helper()
	bin = fakeExecutable(t, t.TempDir(), "php"+v)
	lookups = new(int)
	m.SystemPHP = func() (sysphp.PHP, bool) {
		*lookups++
		return sysphp.PHP{Version: v, Binary: bin}, true
	}
	return bin, lookups
}

func TestAdoptSystem(t *testing.T) {
	t.Run("PHP installed before pvm is current without pvm use", func(t *testing.T) {
		m, _, act := newTestManager(t)
		bin, _ := withSystemPHP(t, m, "8.3")

		a, err := m.Active(t.TempDir(), "")
		if err != nil {
			t.Fatal(err)
		}
		if a.Version != "8.3" || a.Binary != bin || !a.Global() || !a.System {
			t.Errorf("Active = %+v, want global system 8.3 at %s", a, bin)
		}
		if cur, _ := m.Home.Current(); cur != "8.3" {
			t.Errorf("current-version = %q, want 8.3", cur)
		}
		if len(act.activated) != 0 {
			t.Errorf("adoption must not run the activator: %v", act.activated)
		}
	})

	t.Run("looks only once", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		_, lookups := withSystemPHP(t, m, "8.3")

		for range 3 {
			if _, err := m.Active(t.TempDir(), ""); err != nil {
				t.Fatal(err)
			}
		}
		if *lookups != 1 {
			t.Errorf("system php looked up %d times, want 1", *lookups)
		}
	})

	t.Run("keeps the global version already chosen", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		fakeInstall(t, m.Home, "8.4")
		setGlobal(t, m.Home, "8.4")
		_, lookups := withSystemPHP(t, m, "8.3")

		a, err := m.Active(t.TempDir(), "")
		if err != nil {
			t.Fatal(err)
		}
		if a.Version != "8.4" || *lookups != 0 || m.Home.Installed("8.3") {
			t.Errorf("Active = %+v, lookups = %d, 8.3 installed = %v", a, *lookups, m.Home.Installed("8.3"))
		}
	})

	t.Run("reuses the version pvm already installed", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		fakeInstall(t, m.Home, "8.3")
		withSystemPHP(t, m, "8.3")

		a, err := m.Active(t.TempDir(), "")
		if err != nil {
			t.Fatal(err)
		}
		if a.Version != "8.3" || a.System {
			t.Errorf("Active = %+v, want pvm's own 8.3", a)
		}
	})

	t.Run("no PHP on the system", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		m.SystemPHP = func() (sysphp.PHP, bool) { return sysphp.PHP{}, false }

		if _, err := m.Active(t.TempDir(), ""); err != ErrNoActiveVersion {
			t.Fatalf("error = %v, want ErrNoActiveVersion", err)
		}
	})

	t.Run("pvm use of another version keeps the adopted one", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		withSystemPHP(t, m, "8.3")
		fakeInstall(t, m.Home, "8.4")

		if err := m.Use(Target{Arg: "8.4", Version: "8.4"}); err != nil {
			t.Fatal(err)
		}
		if err := m.Use(Target{Arg: "8.3", Version: "8.3"}); err != nil {
			t.Fatalf("adopted 8.3 should be usable: %v", err)
		}
	})

	t.Run("remove forgets it without uninstalling", func(t *testing.T) {
		m, inst, _ := newTestManager(t)
		withSystemPHP(t, m, "8.3")

		r, err := m.Remove("8.3")
		if err != nil {
			t.Fatal(err)
		}
		if !r.System || !r.WasCurrent || inst.removed != nil {
			t.Errorf("removal = %+v, uninstalled = %v", r, inst.removed)
		}
		if _, err := m.Active(t.TempDir(), ""); err != ErrNoActiveVersion {
			t.Errorf("forgotten version must not be adopted again, error = %v", err)
		}
	})

	t.Run("project file still wins", func(t *testing.T) {
		m, _, _ := newTestManager(t)
		withSystemPHP(t, m, "8.3")
		fakeInstall(t, m.Home, "8.2")
		dir := t.TempDir()
		writePHPVersion(t, dir, "8.2")

		a, err := m.Active(dir, "")
		if err != nil {
			t.Fatal(err)
		}
		if a.Version != "8.2" || !strings.HasSuffix(a.Source, filepath.Join(dir, ".php-version")) {
			t.Errorf("Active = %+v, want 8.2 from the project", a)
		}
	})
}
