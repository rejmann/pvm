package sysphp

import "testing"

func TestParseProbe(t *testing.T) {
	tests := []struct {
		out     string
		want    Info
		wantErr bool
	}{
		{out: "\n8.3.12\n1", want: Info{Version: "8.3.12", Zip: true}},
		{out: "\n7.4.33\n0\n", want: Info{Version: "7.4.33"}},
		{out: "PHP Warning:  Module \"x\" is already loaded\n\n8.5.0\r\n1", want: Info{Version: "8.5.0", Zip: true}},
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
