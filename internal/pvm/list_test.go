package pvm

import (
	"reflect"
	"testing"

	"github.com/rejmann/pvm/internal/sysphp"
)

func TestList(t *testing.T) {
	m, _, _ := newTestManager(t)
	fakeInstall(t, m.Home, "8.3")
	fakeInstall(t, m.Home, "8.2")
	setGlobal(t, m.Home, "8.3")
	m.System = func() []sysphp.PHP {
		return []sysphp.PHP{{Version: "8.1", Binary: "/usr/bin/php8.1"}, {Version: "8.3", Binary: "/usr/bin/php8.3"}}
	}

	l, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	want := Listing{
		Managed: []string{"8.2", "8.3"},
		Current: "8.3",
		System:  []sysphp.PHP{{Version: "8.1", Binary: "/usr/bin/php8.1"}},
	}
	if !reflect.DeepEqual(l, want) {
		t.Errorf("List = %+v, want %+v", l, want)
	}
}
