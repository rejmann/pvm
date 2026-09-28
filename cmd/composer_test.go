package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProbe(t *testing.T) {
	tests := []struct {
		out     string
		want    phpInfo
		wantErr bool
	}{
		{out: "\n8.3.12\n1", want: phpInfo{Version: "8.3.12", Zip: true}},
		{out: "\n7.4.33\n0\n", want: phpInfo{Version: "7.4.33"}},
		{out: "PHP Warning:  Module \"x\" is already loaded\n\n8.5.0\r\n1", want: phpInfo{Version: "8.5.0", Zip: true}},
		{out: "8.3.12", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseProbe(tt.out)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseProbe(%q) = %+v, want error", tt.out, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("parseProbe(%q) = (%+v, %v), want %+v", tt.out, got, err, tt.want)
		}
	}
}

func TestHasArchiveTool(t *testing.T) {
	if hasArchiveTool(t.TempDir()) {
		t.Error("found an archive tool in an empty PATH")
	}
	for _, name := range []string{"unzip", "7z"} {
		sys := fakeExecutable(t, t.TempDir(), name)
		if !hasArchiveTool(filepath.Dir(sys)) {
			t.Errorf("%s on PATH not found", name)
		}
	}
}

func TestOfferZipExtensionWithoutInstall(t *testing.T) {
	t.Run("not a terminal", func(t *testing.T) {
		var out bytes.Buffer
		offerZipExtension(t.TempDir(), "8.3", false, strings.NewReader("y\n"), &out)
		if !strings.Contains(out.String(), "PHP 8.3 has no zip extension") ||
			!strings.Contains(out.String(), "in a terminal") {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("declined", func(t *testing.T) {
		var out bytes.Buffer
		offerZipExtension(t.TempDir(), "8.3", true, strings.NewReader("n\n"), &out)
		if strings.Contains(out.String(), "installed") || strings.Contains(out.String(), "Warning") {
			t.Errorf("output = %q", out.String())
		}
	})
}
