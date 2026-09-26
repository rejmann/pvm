package version

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Version
		wantErr bool
	}{
		{in: "8.3", want: Version{Major: 8, Minor: 3}},
		{in: "8.3.30", want: Version{Major: 8, Minor: 3, Patch: 30, hasPatch: true}},
		{in: "7.4.0", want: Version{Major: 7, Minor: 4, Patch: 0, hasPatch: true}},
		{in: "", wantErr: true},
		{in: "8", wantErr: true},
		{in: "8.3.1.2", wantErr: true},
		{in: "8.", wantErr: true},
		{in: "8..1", wantErr: true},
		{in: "08.3", wantErr: true},
		{in: "8.03", wantErr: true},
		{in: "8.x", wantErr: true},
		{in: "-8.3", wantErr: true},
		{in: "lts", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidVersion) {
					t.Fatalf("Parse(%q) error = %v, want ErrInvalidVersion", tt.in, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"8.3", "8.3", 0},
		{"8.3", "8.3.0", 0},
		{"8.3", "8.4", -1},
		{"8.4", "8.3", 1},
		{"7.4.33", "8.0.0", -1},
		{"8.3.10", "8.3.9", 1},
		{"8.3.1", "8.3.10", -1},
	}

	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		if got := a.Compare(b); got != tt.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
