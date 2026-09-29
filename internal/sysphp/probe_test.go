package sysphp

import (
	"slices"
	"testing"
)

func TestParseProbe(t *testing.T) {
	tests := []struct {
		out     string
		want    Info
		wantErr bool
	}{
		{out: "\n8.3.12\nCore,zip,PDO,Zend OPcache,zip", want: Info{Version: "8.3.12", Zip: true,
			Extensions: []string{"core", "opcache", "pdo", "zip"}}},
		{out: "\n7.4.33\nCore,json\n", want: Info{Version: "7.4.33", Extensions: []string{"core", "json"}}},
		{out: "PHP Warning:  Module \"x\" is already loaded\n\n8.5.0\r\nzip", want: Info{Version: "8.5.0", Zip: true,
			Extensions: []string{"zip"}}},
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
		if err != nil || got.Version != tt.want.Version || got.Zip != tt.want.Zip ||
			!slices.Equal(got.Extensions, tt.want.Extensions) {
			t.Errorf("parseProbe(%q) = (%+v, %v), want %+v", tt.out, got, err, tt.want)
		}
	}
}

func TestInfoHas(t *testing.T) {
	i := Info{Extensions: []string{"opcache", "pdo_mysql", "xdebug"}}
	for ext, want := range map[string]bool{"ext-pdo_mysql": true, "Zend OPcache": true, "Xdebug": true, "redis": false} {
		if got := i.Has(ext); got != want {
			t.Errorf("Has(%q) = %v, want %v", ext, got, want)
		}
	}
}
