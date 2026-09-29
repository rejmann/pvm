package pvm

import (
	"errors"
	"strings"
	"testing"
)

func TestUse(t *testing.T) {
	t.Run("activates the installed binary", func(t *testing.T) {
		m, _, act := newTestManager(t)
		fakeInstall(t, m.Home, "8.2")
		bin, _ := m.Home.Binary("8.2")

		if err := m.Use(Target{Arg: "8.2", Version: "8.2"}); err != nil {
			t.Fatal(err)
		}
		if strings.Join(act.activated, ",") != "8.2="+bin {
			t.Errorf("activated = %v", act.activated)
		}
	})

	t.Run("not installed", func(t *testing.T) {
		m, _, act := newTestManager(t)

		err := m.Use(Target{Arg: "lts", Version: "8.4", Alias: true})
		if err == nil || !strings.Contains(err.Error(), "8.4 (lts) not installed — run: pvm install lts") {
			t.Fatalf("error = %v", err)
		}
		if act.activated != nil {
			t.Errorf("activated = %v", act.activated)
		}
	})

	t.Run("activation fails", func(t *testing.T) {
		m, _, act := newTestManager(t)
		fakeInstall(t, m.Home, "8.2")
		boom := errors.New("activation failed")
		act.err = boom

		if err := m.Use(Target{Version: "8.2"}); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}

func TestProjectVersion(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := ProjectVersion(dir); err == nil || !strings.Contains(err.Error(), "no .php-version found") {
		t.Fatalf("no file: error = %v", err)
	}

	writePHPVersion(t, dir, "8.2")
	if v, _, err := ProjectVersion(dir); err != nil || v != "8.2" {
		t.Fatalf("ProjectVersion = (%q, %v)", v, err)
	}
}
