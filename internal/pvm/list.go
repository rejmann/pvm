package pvm

import (
	"fmt"

	"github.com/rejmann/pvm/internal/sysphp"
)

// Listing is what pvm list shows.
type Listing struct {
	Managed []string     // installed by pvm, oldest first
	Current string       // the global version, if any
	System  []sysphp.PHP // found on the system, excluding versions pvm manages
}

// List returns the installed versions and the system PHP pvm does not manage.
func (m *Manager) List() (Listing, error) {
	managed, err := m.Home.Versions()
	if err != nil {
		return Listing{}, fmt.Errorf("read installed versions: %w", err)
	}
	l := Listing{Managed: managed}
	l.Current, _ = m.Home.Current()

	isManaged := map[string]bool{}
	for _, v := range managed {
		isManaged[v] = true
	}
	if m.System != nil {
		for _, s := range m.System() {
			if !isManaged[s.Version] {
				l.System = append(l.System, s)
			}
		}
	}
	return l, nil
}
