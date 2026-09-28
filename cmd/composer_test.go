package cmd

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseProbe(t *testing.T) {
	tests := []struct {
		out     string
		want    phpInfo
		wantErr bool
	}{
		{out: "\n8.3.12\nmissing:", want: phpInfo{Version: "8.3.12"}},
		{out: "\n7.4.33\nmissing:xml,zip\n", want: phpInfo{Version: "7.4.33", Missing: []string{"xml", "zip"}}},
		{out: "PHP Warning:  Module \"x\" is already loaded\n\n8.5.0\r\nmissing:mbstring", want: phpInfo{Version: "8.5.0", Missing: []string{"mbstring"}}},
		{out: "8.3.12", wantErr: true},
		{out: "\n8.3.12\n1", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseProbe(tt.out)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseProbe(%q) = %+v, want error", tt.out, got)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseProbe(%q) = (%+v, %v), want %+v", tt.out, got, err, tt.want)
		}
	}
}

func TestNeededExtensions(t *testing.T) {
	missing := []string{"xml", "zip"}
	if got := neededExtensions(missing, t.TempDir()); !reflect.DeepEqual(got, missing) {
		t.Errorf("without unzip: got %v, want %v", got, missing)
	}
	unzip := fakeExecutable(t, t.TempDir(), "unzip")
	if got := neededExtensions(missing, filepath.Dir(unzip)); !reflect.DeepEqual(got, []string{"xml"}) {
		t.Errorf("with unzip: got %v, want [xml]", got)
	}
	if got := neededExtensions([]string{"zip"}, filepath.Dir(unzip)); len(got) != 0 {
		t.Errorf("only zip, with unzip: got %v, want none", got)
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

func TestOfferExtensionsWithoutInstall(t *testing.T) {
	t.Run("not a terminal", func(t *testing.T) {
		var out bytes.Buffer
		offerExtensions(t.TempDir(), "8.3", []string{"xml", "zip"}, false, strings.NewReader("y\n"), &out)
		if !strings.Contains(out.String(), "PHP 8.3 is missing the xml, zip extension(s)") ||
			!strings.Contains(out.String(), "in a terminal") {
			t.Errorf("output = %q", out.String())
		}
	})

	t.Run("declined", func(t *testing.T) {
		var out bytes.Buffer
		offerExtensions(t.TempDir(), "8.3", []string{"xml"}, true, strings.NewReader("n\n"), &out)
		if strings.Contains(out.String(), "installed") || strings.Contains(out.String(), "Warning") {
			t.Errorf("output = %q", out.String())
		}
	})
}
