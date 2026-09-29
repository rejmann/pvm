package pvm

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/composer"
)

func TestRemove(t *testing.T) {
	t.Run("removes inactive version, its metadata and Composer", func(t *testing.T) {
		m, inst, act := newTestManager(t)
		fakeInstall(t, m.Home, "8.2")
		mkdirAll(t, composer.VersionDir(m.Home.ComposerDir(), "8.2"))
		mkdirAll(t, composer.VersionDir(m.Home.ComposerDir(), "8.3"))

		r, err := m.Remove("8.2")
		if err != nil {
			t.Fatal(err)
		}
		if r.WasCurrent || r.ComposerErr != nil || act.deactivated != 0 {
			t.Errorf("removal = %+v, deactivated = %d", r, act.deactivated)
		}
		if strings.Join(inst.removed, ",") != "8.2" {
			t.Errorf("remover got %v", inst.removed)
		}
		if _, err := os.Stat(m.Home.VersionDir("8.2")); !os.IsNotExist(err) {
			t.Errorf("version dir should be gone, stat err = %v", err)
		}
		if _, err := os.Stat(composer.VersionDir(m.Home.ComposerDir(), "8.2")); !os.IsNotExist(err) {
			t.Errorf("Composer of 8.2 should be gone, stat err = %v", err)
		}
		if _, err := os.Stat(composer.VersionDir(m.Home.ComposerDir(), "8.3")); err != nil {
			t.Errorf("Composer of 8.3 must stay: %v", err)
		}
	})

	t.Run("deactivates the global version", func(t *testing.T) {
		m, _, act := newTestManager(t)
		fakeInstall(t, m.Home, "8.2")
		setGlobal(t, m.Home, "8.2")

		r, err := m.Remove("8.2")
		if err != nil {
			t.Fatal(err)
		}
		if !r.WasCurrent || act.deactivated != 1 {
			t.Errorf("removal = %+v, deactivated = %d", r, act.deactivated)
		}
	})

	for _, tt := range []struct{ name, v, wantErr string }{
		{name: "invalid version", v: "lts", wantErr: "invalid version"},
		{name: "not installed", v: "8.1", wantErr: "not installed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, inst, _ := newTestManager(t)
			if _, err := m.Remove(tt.v); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %s", err, tt.wantErr)
			}
			if inst.removed != nil {
				t.Errorf("remover ran: %v", inst.removed)
			}
		})
	}

	t.Run("keeps metadata when remover fails", func(t *testing.T) {
		m, inst, _ := newTestManager(t)
		fakeInstall(t, m.Home, "8.2")
		boom := errors.New("permission denied")
		inst.err = boom

		if _, err := m.Remove("8.2"); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
		if !m.Home.Installed("8.2") {
			t.Error("version metadata should survive a failed removal")
		}
	})
}
