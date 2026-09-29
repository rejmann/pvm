package pvm

import (
	"errors"
	"strings"
	"testing"
)

func TestInstall(t *testing.T) {
	t.Run("installs and announces", func(t *testing.T) {
		m, inst, _ := newTestManager(t)
		started := false

		if err := m.Install(Target{Arg: "8.3", Version: "8.3"}, func() { started = true }); err != nil {
			t.Fatal(err)
		}
		if !started || strings.Join(inst.installed, ",") != "8.3" || !m.Home.Installed("8.3") {
			t.Errorf("started = %v, installed = %v", started, inst.installed)
		}
	})

	t.Run("rejects already installed", func(t *testing.T) {
		m, inst, _ := newTestManager(t)
		fakeInstall(t, m.Home, "8.4")

		err := m.Install(Target{Arg: "lts", Version: "8.4", Alias: true}, func() { t.Error("must not start") })
		if err == nil || !strings.Contains(err.Error(), "8.4 (lts) already installed") {
			t.Fatalf("error = %v", err)
		}
		if inst.installed != nil {
			t.Errorf("installer ran: %v", inst.installed)
		}
	})

	t.Run("propagates installer error", func(t *testing.T) {
		m, inst, _ := newTestManager(t)
		boom := errors.New("apt failed")
		inst.err = boom

		if err := m.Install(Target{Version: "8.3"}, nil); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want wrapped %v", err, boom)
		}
	})
}

func TestTarget(t *testing.T) {
	m, _, _ := newTestManager(t)

	got, err := m.Target("8.3")
	if err != nil || got != (Target{Arg: "8.3", Version: "8.3"}) || got.String() != "8.3" {
		t.Fatalf("Target(8.3) = (%+v, %v)", got, err)
	}

	m.LTS = fakeResolver{v: "8.4"}
	got, err = m.Target("lts")
	if err != nil || got.Version != "8.4" || got.String() != "8.4 (lts)" {
		t.Fatalf("Target(lts) = (%+v, %v)", got, err)
	}

	boom := errors.New("offline")
	m.LTS = fakeResolver{err: boom}
	if _, err := m.Target("lts"); !errors.Is(err, boom) {
		t.Errorf("resolver error = %v, want wrapped %v", err, boom)
	}

	for _, bad := range []string{"8", "8.x"} {
		if _, err := m.Target(bad); err == nil || !strings.Contains(err.Error(), "invalid version") {
			t.Errorf("Target(%q) error = %v, want invalid version", bad, err)
		}
	}
}
