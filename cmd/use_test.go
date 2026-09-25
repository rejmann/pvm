package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/rejmann/pvm/internal/version"
)

// Only the validation paths are covered here: a successful switch calls
// symlink.SetCurrent, which runs sudo update-alternatives on Linux.
func TestUseVersionErrors(t *testing.T) {
	boom := errors.New("offline")

	tests := []struct {
		name    string
		arg     string
		r       version.Resolver
		wantErr string
		wantIs  error
	}{
		{name: "invalid version", arg: "8", wantErr: "invalid version"},
		{name: "not installed", arg: "8.3", wantErr: "8.3 not installed — run: pvm install 8.3"},
		{name: "lts not installed", arg: "lts", r: fakeResolver{v: "8.4"}, wantErr: "8.4 (lts) not installed — run: pvm install lts"},
		{name: "resolver error", arg: "lts", r: fakeResolver{err: boom}, wantIs: boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newManager(t)
			fakeInstall(t, m, "8.2")

			var r version.Resolver = failResolver{t}
			if tt.r != nil {
				r = tt.r
			}

			err := useVersion(tt.arg, m, r, &bytes.Buffer{})
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("error = %v, want wrapped %v", err, tt.wantIs)
			}
		})
	}
}
