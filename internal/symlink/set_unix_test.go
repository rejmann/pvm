//go:build linux || darwin

package symlink

import (
	"errors"
	"testing"
)

func TestCurrentVersionRoundTrip(t *testing.T) {
	base := t.TempDir()
	if err := writeCurrentVersion(base, "8.2"); err != nil {
		t.Fatal(err)
	}
	got, err := GetCurrent(base)
	if err != nil || got != "8.2" {
		t.Fatalf("GetCurrent = (%q, %v), want (\"8.2\", nil)", got, err)
	}

	if err := removeCurrentVersion(base); err != nil {
		t.Fatal(err)
	}
	if _, err := GetCurrent(base); !errors.Is(err, ErrNoCurrentVersion) {
		t.Errorf("after remove, GetCurrent error = %v, want ErrNoCurrentVersion", err)
	}
	// Removing again is a no-op.
	if err := removeCurrentVersion(base); err != nil {
		t.Errorf("second removeCurrentVersion = %v, want nil", err)
	}
}
